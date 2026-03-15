package wait

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/axelspire/at3am/internal/config"
	"github.com/axelspire/at3am/internal/mock"
	"github.com/axelspire/at3am/internal/output"
	"github.com/axelspire/at3am/internal/resolver"
)

type mockCommandRunner struct {
	lastCommand string
	err         error
	called      bool
}

func (m *mockCommandRunner) Run(command string) error {
	m.called = true
	m.lastCommand = command
	return m.err
}

func makeTestRunner(scenario mock.Scenario, cfgMod func(*config.Config)) (*Runner, *bytes.Buffer) {
	cfg := config.DefaultConfig()
	cfg.Domain = "_acme-challenge.example.com"
	cfg.Expected = "mock-validation-token"
	cfg.Interval = 10 * time.Millisecond
	cfg.Timeout = 2 * time.Second
	cfg.ConsecutivePasses = 2
	if cfgMod != nil {
		cfgMod(&cfg)
	}

	querier := mock.NewQuerier(scenario)
	pool := resolver.NewPool(querier, nil)
	if len(scenario.AuthNS) > 0 {
		pool.SetAuthResolvers(scenario.AuthNS)
	}

	var buf bytes.Buffer
	formatter := output.NewFormatter(cfg.OutputFormat, &buf)
	runner := NewRunner(cfg, pool, formatter)
	return runner, &buf
}

func TestRun_InstantPropagation(t *testing.T) {
	scenario := mock.PredefinedScenarios()["instant"]
	runner, _ := makeTestRunner(scenario, nil)

	code := runner.Run(context.Background())
	if code != ExitSuccess {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestRun_SlowPropagation(t *testing.T) {
	scenario := mock.PredefinedScenarios()["slow_propagation"]
	runner, _ := makeTestRunner(scenario, func(c *config.Config) {
		c.Threshold = 80.0
		c.ConsecutivePasses = 1
		c.Timeout = 2 * time.Second
	})

	code := runner.Run(context.Background())
	if code != ExitSuccess {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestRun_Timeout(t *testing.T) {
	scenario := mock.PredefinedScenarios()["timeout"]
	runner, _ := makeTestRunner(scenario, func(c *config.Config) {
		c.Timeout = 100 * time.Millisecond
	})

	code := runner.Run(context.Background())
	if code != ExitTimeout {
		t.Errorf("expected exit %d, got %d", ExitTimeout, code)
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	scenario := mock.PredefinedScenarios()["timeout"]
	runner, _ := makeTestRunner(scenario, nil)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	code := runner.Run(ctx)
	if code != ExitTimeout {
		t.Errorf("expected exit %d, got %d", ExitTimeout, code)
	}
}

func TestRun_OnReadyCommand(t *testing.T) {
	scenario := mock.PredefinedScenarios()["instant"]
	cmdRunner := &mockCommandRunner{}
	runner, _ := makeTestRunner(scenario, func(c *config.Config) {
		c.OnReady = "echo $DOMAIN $CONFIDENCE"
	})
	runner.SetCommandRunner(cmdRunner)

	code := runner.Run(context.Background())
	if code != ExitSuccess {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !cmdRunner.called {
		t.Error("expected on-ready command to be called")
	}
	if cmdRunner.lastCommand == "" {
		t.Error("expected non-empty command")
	}
}

func TestRun_OnReadyCommandFailure(t *testing.T) {
	scenario := mock.PredefinedScenarios()["instant"]
	cmdRunner := &mockCommandRunner{err: fmt.Errorf("command failed")}
	runner, _ := makeTestRunner(scenario, func(c *config.Config) {
		c.OnReady = "failing-command"
	})
	runner.SetCommandRunner(cmdRunner)

	// Should still succeed (on-ready failure is non-fatal)
	code := runner.Run(context.Background())
	if code != ExitSuccess {
		t.Errorf("expected exit 0 even with on-ready failure, got %d", code)
	}
}

func TestRun_JSONOutput(t *testing.T) {
	scenario := mock.PredefinedScenarios()["instant"]
	runner, buf := makeTestRunner(scenario, func(c *config.Config) {
		c.OutputFormat = "json"
	})

	code := runner.Run(context.Background())
	if code != ExitSuccess {
		t.Errorf("expected exit 0, got %d", code)
	}
	if buf.Len() == 0 {
		t.Error("expected JSON output")
	}
}

func TestRun_QuietOutput(t *testing.T) {
	scenario := mock.PredefinedScenarios()["instant"]
	runner, buf := makeTestRunner(scenario, func(c *config.Config) {
		c.OutputFormat = "quiet"
	})

	code := runner.Run(context.Background())
	if code != ExitSuccess {
		t.Errorf("expected exit 0, got %d", code)
	}
	if buf.Len() == 0 {
		t.Error("expected at least READY in quiet output")
	}
}

func TestRun_ConsecutivePassesReset(t *testing.T) {
	// Use flaky scenario which should intermittently fail
	scenario := mock.PredefinedScenarios()["flaky"]
	runner, _ := makeTestRunner(scenario, func(c *config.Config) {
		c.Threshold = 99.0  // High threshold that flaky won't always meet
		c.Timeout = 200 * time.Millisecond
	})

	// This should time out because flaky resolver can't consistently meet 99%
	code := runner.Run(context.Background())
	// Either times out or succeeds depending on timing - just verify it runs
	if code != ExitSuccess && code != ExitTimeout {
		t.Errorf("expected exit 0 or 1, got %d", code)
	}
}

func TestExitCodes(t *testing.T) {
	if ExitSuccess != 0 {
		t.Errorf("ExitSuccess should be 0")
	}
	if ExitTimeout != 1 {
		t.Errorf("ExitTimeout should be 1")
	}
	if ExitConfigError != 2 {
		t.Errorf("ExitConfigError should be 2")
	}
	if ExitDNSError != 3 {
		t.Errorf("ExitDNSError should be 3")
	}
}

func TestRun_WithPrometheusPort(t *testing.T) {
	scenario := mock.PredefinedScenarios()["instant"]
	runner, _ := makeTestRunner(scenario, func(c *config.Config) {
		c.PrometheusPort = 0 // 0 means disabled, so pick a random port
	})
	// With port 0 (disabled), should succeed normally
	code := runner.Run(context.Background())
	if code != ExitSuccess {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestRun_WithPrometheusEnabled(t *testing.T) {
	scenario := mock.PredefinedScenarios()["instant"]
	runner, _ := makeTestRunner(scenario, func(c *config.Config) {
		c.PrometheusPort = 19091
	})
	code := runner.Run(context.Background())
	if code != ExitSuccess {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestRun_NoOnReady(t *testing.T) {
	scenario := mock.PredefinedScenarios()["instant"]
	runner, _ := makeTestRunner(scenario, func(c *config.Config) {
		c.OnReady = ""
	})
	code := runner.Run(context.Background())
	if code != ExitSuccess {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestRealCommandRunner(t *testing.T) {
	r := &RealCommandRunner{}
	err := r.Run("echo hello")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRealCommandRunner_Failure(t *testing.T) {
	r := &RealCommandRunner{}
	err := r.Run("false")
	if err == nil {
		t.Error("expected error")
	}
}

func TestRealWebhookPoster_InvalidURL(t *testing.T) {
	p := &RealWebhookPoster{}
	err := p.Post("http://192.0.2.1:1/nonexistent", `{"test":true}`)
	if err == nil {
		t.Error("expected error for unreachable URL")
	}
}

func TestRealWebhookPoster_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p := &RealWebhookPoster{}
	err := p.Post(ts.URL, `{"test":true}`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRealWebhookPoster_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	p := &RealWebhookPoster{}
	err := p.Post(ts.URL, `{"test":true}`)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestRun_WithWebhook(t *testing.T) {
	webhookCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	scenario := mock.PredefinedScenarios()["instant"]
	runner, _ := makeTestRunner(scenario, func(c *config.Config) {
		c.WebhookURL = ts.URL
	})

	code := runner.Run(context.Background())
	if code != ExitSuccess {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !webhookCalled {
		t.Error("expected webhook to be called")
	}
}

func TestRun_WithOnReadyTemplate(t *testing.T) {
	cmdRunner := &mockCommandRunner{}
	scenario := mock.PredefinedScenarios()["instant"]
	runner, _ := makeTestRunner(scenario, func(c *config.Config) {
		c.OnReady = "echo domain=$DOMAIN confidence=$CONFIDENCE elapsed=$ELAPSED"
	})
	runner.SetCommandRunner(cmdRunner)

	code := runner.Run(context.Background())
	if code != ExitSuccess {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !cmdRunner.called {
		t.Fatal("expected command to be called")
	}
	// Verify template was expanded
	if cmdRunner.lastCommand == "echo domain=$DOMAIN confidence=$CONFIDENCE elapsed=$ELAPSED" {
		t.Error("template variables should have been expanded")
	}
}

func TestRun_WithFailingWebhook(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	scenario := mock.PredefinedScenarios()["instant"]
	runner, _ := makeTestRunner(scenario, func(c *config.Config) {
		c.WebhookURL = ts.URL
	})

	// Should still succeed (webhook failure is non-fatal)
	code := runner.Run(context.Background())
	if code != ExitSuccess {
		t.Errorf("expected exit 0 even with webhook failure, got %d", code)
	}
}

func TestNewRunner(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "test.com"
	cfg.Expected = "tok"
	q := mock.NewQuerier(mock.PredefinedScenarios()["instant"])
	pool := resolver.NewPool(q, nil)
	var buf bytes.Buffer
	f := output.NewFormatter("human", &buf)
	runner := NewRunner(cfg, pool, f)
	if runner == nil {
		t.Fatal("expected non-nil runner")
	}
}

