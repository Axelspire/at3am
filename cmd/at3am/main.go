package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/axelspire/at3am/internal/config"
	"github.com/axelspire/at3am/internal/log"
	"github.com/axelspire/at3am/internal/output"
	"github.com/axelspire/at3am/internal/resolver"
	"github.com/axelspire/at3am/internal/wait"
	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "at3am",
	Short: "Intelligent DNS-01 validation for ACME clients",
	Long:  "at3am watches global DNS resolvers and signals when a DNS-01 TXT record has propagated with sufficient confidence.",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("at3am version %s (commit %s, built %s)\n", version, commit, buildTime)
	},
}

var waitCmd = &cobra.Command{
	Use:   "wait",
	Short: "Wait for DNS propagation",
	Long:  "Poll global DNS resolvers until a TXT record has propagated with sufficient confidence.",
	RunE:  runWait,
}

func init() {
	rootCmd.AddCommand(versionCmd, waitCmd)

	// Required flags
	waitCmd.Flags().StringP("domain", "d", "", "Domain to check (required)")
	waitCmd.Flags().StringP("expected", "e", "", "Expected TXT record value (required)")
	waitCmd.MarkFlagRequired("domain")
	waitCmd.MarkFlagRequired("expected")

	// Timing
	waitCmd.Flags().DurationP("timeout", "t", 5*time.Minute, "Maximum time to wait")
	waitCmd.Flags().DurationP("interval", "i", 5*time.Second, "Poll interval")

	// Profiles
	waitCmd.Flags().StringP("profile", "p", "default", "Profile: default, strict, or fast")

	// Output
	waitCmd.Flags().StringP("output", "o", "text", "Output format: text, json, or quiet")

	// Logging
	waitCmd.Flags().StringP("log-level", "l", "warn", "Log level: debug, info, warn, error")
	waitCmd.Flags().String("log-file", "", "Log file path")

	// Resolvers
	waitCmd.Flags().StringSlice("resolvers", nil, "Custom resolver addresses")

	// DNSSEC
	waitCmd.Flags().Bool("dnssec-validate", false, "Enable DNSSEC validation")

	// Challenge type
	waitCmd.Flags().String("challenge-type", "dns-01", "Challenge type: dns-01 or persist")

	// Automation
	waitCmd.Flags().String("on-ready", "", "Command to run when ready")
	waitCmd.Flags().String("webhook", "", "Webhook URL to POST when ready")

	// Prometheus
	waitCmd.Flags().Int("prometheus-port", 0, "Prometheus metrics port")

	// Mock
	waitCmd.Flags().Bool("mock", false, "Enable mock DNS mode")
	waitCmd.Flags().String("mock-scenario", "instant", "Mock scenario: instant, slow_propagation, timeout, flaky, partial")
}

func runWait(cmd *cobra.Command, args []string) error {
	domain, _ := cmd.Flags().GetString("domain")
	expected, _ := cmd.Flags().GetString("expected")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	interval, _ := cmd.Flags().GetDuration("interval")
	profile, _ := cmd.Flags().GetString("profile")
	outputFormat, _ := cmd.Flags().GetString("output")
	logLevelStr, _ := cmd.Flags().GetString("log-level")
	logFile, _ := cmd.Flags().GetString("log-file")
	customResolvers, _ := cmd.Flags().GetStringSlice("resolvers")
	dnssecValidate, _ := cmd.Flags().GetBool("dnssec-validate")
	challengeTypeStr, _ := cmd.Flags().GetString("challenge-type")
	onReady, _ := cmd.Flags().GetString("on-ready")
	webhook, _ := cmd.Flags().GetString("webhook")
	prometheusPort, _ := cmd.Flags().GetInt("prometheus-port")
	mockMode, _ := cmd.Flags().GetBool("mock")
	mockScenario, _ := cmd.Flags().GetString("mock-scenario")

	// Parse log level
	logLevel, err := log.ParseLevel(logLevelStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}

	// Initialize logger
	teardown, err := log.Init(logLevel, logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}
	defer teardown()

	// Build config
	cfg := config.Config{
		Domain:         domain,
		Expected:       expected,
		Timeout:        timeout,
		Interval:       interval,
		Profile:        config.Profile(profile),
		OutputFormat:   outputFormat,
		Resolvers:      customResolvers,
		DNSSECValidate: dnssecValidate,
		ChallengeType:  config.ChallengeType(challengeTypeStr),
		OnReady:        onReady,
		WebhookURL:     webhook,
		PrometheusPort: prometheusPort,
		MockMode:       mockMode,
		MockScenario:   mockScenario,
		LogLevel:       logLevelStr,
		LogFile:        logFile,
	}

	// Apply profile
	if err := cfg.ApplyProfile(config.Profile(profile)); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}

	// Create real DNS querier
	querier := resolver.New(2 * time.Second)
	querier.SetDNSSECValidate(dnssecValidate)

	// Create resolver pool
	pool := resolver.NewPool(querier, customResolvers)

	// Create formatter
	formatter := output.NewFormatter(outputFormat, os.Stdout)

	// Create runner
	runner := wait.NewRunner(cfg, pool, formatter)

	// Run
	ctx := context.Background()
	exitCode := runner.Run(ctx)
	os.Exit(exitCode)
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

