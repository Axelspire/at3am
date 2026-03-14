// Package wait implements the core polling loop for DNS propagation monitoring.
package wait

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/axelspire/at3am/internal/confidence"
	"github.com/axelspire/at3am/internal/config"
	"github.com/axelspire/at3am/internal/diagnostics"
	logger "github.com/axelspire/at3am/internal/log"
	"github.com/axelspire/at3am/internal/output"
	"github.com/axelspire/at3am/internal/resolver"
	"github.com/axelspire/at3am/internal/ttl"
)

// ExitCode constants.
const (
	ExitSuccess       = 0
	ExitTimeout       = 1
	ExitConfigError   = 2
	ExitDNSError      = 3
)

// Runner orchestrates the polling loop.
type Runner struct {
	cfg          config.Config
	pool         *resolver.Pool
	engine       *confidence.Engine
	ttlAnalyser  *ttl.Analyser
	diagEngine   *diagnostics.Engine
	formatter    *output.Formatter
	metrics      *output.Metrics
	metricsSrv   *http.Server
	commandRunner CommandRunner
}

// CommandRunner is an interface for running shell commands (for testability).
type CommandRunner interface {
	Run(command string) error
}

// RealCommandRunner executes real shell commands.
type RealCommandRunner struct{}

// Run executes a shell command.
func (r *RealCommandRunner) Run(command string) error {
	cmd := exec.Command("sh", "-c", command)
	return cmd.Run()
}

// WebhookPoster is an interface for posting webhooks.
type WebhookPoster interface {
	Post(url string, body string) error
}

// RealWebhookPoster posts real HTTP webhooks.
type RealWebhookPoster struct{}

// Post sends an HTTP POST to the given URL.
func (p *RealWebhookPoster) Post(url string, body string) error {
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

// NewRunner creates a new Runner.
func NewRunner(cfg config.Config, pool *resolver.Pool, formatter *output.Formatter) *Runner {
	// Use profile-based engine if a profile is set, otherwise use legacy weights
	var engine *confidence.Engine
	if cfg.Profile != "" {
		profileType := confidence.ProfileFromString(string(cfg.Profile))
		engine = confidence.NewEngineWithProfile(profileType)
	} else {
		engine = confidence.NewEngine(cfg.AuthWeight, cfg.PublicWeight, cfg.Threshold)
	}

	return &Runner{
		cfg:         cfg,
		pool:        pool,
		engine:      engine,
		ttlAnalyser: ttl.NewAnalyser(),
		diagEngine:  diagnostics.NewEngine(),
		formatter:   formatter,
		metrics:     output.NewMetrics(),
		commandRunner: &RealCommandRunner{},
	}
}

// SetCommandRunner sets a custom command runner (for testing).
func (r *Runner) SetCommandRunner(cr CommandRunner) {
	r.commandRunner = cr
}

// Run executes the polling loop. Returns the exit code.
func (r *Runner) Run(ctx context.Context) int {
	// Start Prometheus metrics server if configured
	if r.cfg.PrometheusPort > 0 {
		srv, err := output.StartMetricsServer(r.cfg.PrometheusPort, r.metrics)
		if err != nil {
			r.formatter.EmitFinalResult(output.FinalResult{
				Success: false, Domain: r.cfg.Domain,
				Message: fmt.Sprintf("failed to start metrics server: %v", err),
			})
			return ExitConfigError
		}
		r.metricsSrv = srv
		defer srv.Close()
	}

	deadline := time.Now().Add(r.cfg.Timeout)
	attempt := 0
	consecutivePasses := 0
	startTime := time.Now()

	for {
		if time.Now().After(deadline) {
			elapsed := time.Since(startTime)
			logger.Warn("done | status=TIMEOUT elapsed=%s attempts=%d", elapsed.Round(time.Millisecond), attempt)
			r.formatter.EmitFinalResult(output.FinalResult{
				Success: false, Domain: r.cfg.Domain,
				Elapsed: elapsed, Attempts: attempt,
				Message: "Timeout exceeded before propagation was confirmed",
			})
			return ExitTimeout
		}

		select {
		case <-ctx.Done():
			return ExitTimeout
		default:
		}

		attempt++
		logger.Debug("poll #%d starting | challenge=%s", attempt, r.cfg.ChallengeType)

		// dns-01: auth-first gating prevents negative-caching on public resolvers
		// before the authoritative source has confirmed the record.
		// persist: record is pre-provisioned; query all resolvers simultaneously.
		var results []resolver.Result
		if r.cfg.ChallengeType == config.ChallengeTypePersist {
			results = r.pool.QueryAll(ctx, r.cfg.Domain)
		} else {
			results = r.pool.QueryAuthFirst(ctx, r.cfg.Domain)
		}
		score := r.engine.Calculate(results, r.cfg.Expected)
		ttlReport := r.ttlAnalyser.Analyse(results)
		diagnosis := r.diagEngine.Diagnose(results, score, ttlReport)

		logger.Info("poll #%d | elapsed=%s auth=%d/%d(need %d) pub=%d/%d(need %d) ready=%v scenario=%s",
			attempt, time.Since(startTime).Round(time.Millisecond),
			score.AuthFound, score.AuthTotal, score.AuthRequired,
			score.PublicFound, score.PublicTotal, score.PublicRequired,
			score.Ready, diagnosis.Scenario)
		if score.DNSSECChecked {
			logger.Debug("poll #%d | dnssec=%d/%d authenticated (AD bit)", attempt, score.DNSSECValidCount, score.DNSSECTotal)
		}

		if score.Ready {
			consecutivePasses++
			logger.Debug("poll #%d | consecutive=%d/%d", attempt, consecutivePasses, r.cfg.ConsecutivePasses)
		} else {
			if consecutivePasses > 0 {
				logger.Debug("poll #%d | consecutive reset (was %d)", attempt, consecutivePasses)
			}
			consecutivePasses = 0
		}

		status := output.PollStatus{
			Domain: r.cfg.Domain, Expected: r.cfg.Expected,
			Attempt: attempt, Score: score,
			TTLReport: ttlReport, Diagnosis: diagnosis,
			Elapsed: time.Since(startTime),
			ConsecPasses:   consecutivePasses,
			RequiredPasses: r.cfg.ConsecutivePasses,
			Ready: consecutivePasses >= r.cfg.ConsecutivePasses,
		}

		r.metrics.Update(status)
		r.formatter.EmitPollStatus(status)

		if consecutivePasses >= r.cfg.ConsecutivePasses {
			elapsed := time.Since(startTime)
			logger.Info("done | status=READY elapsed=%s attempts=%d auth=%d/%d pub=%d/%d",
				elapsed.Round(time.Millisecond), attempt,
				score.AuthFound, score.AuthTotal,
				score.PublicFound, score.PublicTotal)
			r.formatter.EmitFinalResult(output.FinalResult{
				Success: true, Domain: r.cfg.Domain,
				Confidence: score.Overall, Elapsed: elapsed, Attempts: attempt,
			})
			r.executeOnReady(elapsed, score.Overall)
			return ExitSuccess
		}

		time.Sleep(r.cfg.Interval)
	}
}

func (r *Runner) executeOnReady(elapsed time.Duration, conf float64) {
	if r.cfg.OnReady != "" {
		expanded := output.ExpandTemplate(r.cfg.OnReady, r.cfg.Domain, conf, elapsed)
		if err := r.commandRunner.Run(expanded); err != nil {
			fmt.Printf("on-ready command failed: %v\n", err)
		}
	}
	if r.cfg.WebhookURL != "" {
		body := fmt.Sprintf(`{"domain":%q,"confidence":%.1f,"elapsed":%q}`,
			r.cfg.Domain, conf, elapsed)
		poster := &RealWebhookPoster{}
		if err := poster.Post(r.cfg.WebhookURL, body); err != nil {
			fmt.Printf("webhook failed: %v\n", err)
		}
	}
}

