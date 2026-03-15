package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/axelspire/at3am/internal/config"
	"github.com/axelspire/at3am/internal/log"
	"github.com/axelspire/at3am/internal/output"
	"github.com/axelspire/at3am/internal/provider"
	"github.com/axelspire/at3am/internal/resolver"
	"github.com/axelspire/at3am/internal/wait"
	"github.com/libdns/libdns"
	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "at3am-hook",
	Short: "Certbot integration hook for at3am",
	Long:  "at3am-hook is a Certbot manual auth hook that creates DNS records, waits for propagation, and cleans up.",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("at3am-hook version %s (commit %s, built %s)\n", version, commit, buildTime)
	},
}

var manualAuthCmd = &cobra.Command{
	Use:   "manual-auth",
	Short: "Certbot manual auth hook",
	Long:  "Create DNS record and wait for propagation. Called by Certbot during validation.",
	RunE:  runManualAuth,
}

var manualCleanupCmd = &cobra.Command{
	Use:   "manual-cleanup",
	Short: "Certbot manual cleanup hook",
	Long:  "Delete DNS record after validation. Called by Certbot after validation completes.",
	RunE:  runManualCleanup,
}

func init() {
	rootCmd.AddCommand(versionCmd, manualAuthCmd, manualCleanupCmd)

	// Flags for manual-auth
	manualAuthCmd.Flags().String("provider", "", "DNS provider (auto-detected if not set)")
	manualAuthCmd.Flags().String("creds", "", "Path to credentials YAML file")
	manualAuthCmd.Flags().String("profile", "default", "Profile: default, strict, or fast")
	manualAuthCmd.Flags().String("log-level", "warn", "Log level: debug, info, warn, error")
	manualAuthCmd.Flags().String("log-file", "", "Log file path")

	// Flags for manual-cleanup
	manualCleanupCmd.Flags().String("provider", "", "DNS provider (auto-detected if not set)")
	manualCleanupCmd.Flags().String("creds", "", "Path to credentials YAML file")
	manualCleanupCmd.Flags().String("log-level", "warn", "Log level: debug, info, warn, error")
	manualCleanupCmd.Flags().String("log-file", "", "Log file path")
}

func runManualAuth(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// 1. Required Certbot env vars
	domain := os.Getenv("CERTBOT_DOMAIN")
	validation := os.Getenv("CERTBOT_VALIDATION")
	if domain == "" || validation == "" {
		return fmt.Errorf("CERTBOT_DOMAIN and CERTBOT_VALIDATION environment variables are required")
	}
	challengeFQDN := "_acme-challenge." + domain

	// 2. Resolve config from flags + env vars (flags take precedence over env)
	providerName, _ := cmd.Flags().GetString("provider")
	if providerName == "" {
		providerName = os.Getenv("AT3AM_DNS_PROVIDER")
	}
	credsPath, _ := cmd.Flags().GetString("creds")
	if credsPath == "" {
		credsPath = os.Getenv("AT3AM_DNS_CREDS")
	}
	skipDNS := os.Getenv("AT3AM_SKIP_DNS") == "1"
	profileStr, _ := cmd.Flags().GetString("profile")
	if e := os.Getenv("AT3AM_PROFILE"); e != "" {
		profileStr = e
	}
	outputFormat := os.Getenv("AT3AM_OUTPUT")
	if outputFormat == "" {
		outputFormat = "quiet"
	}
	logLevelStr, _ := cmd.Flags().GetString("log-level")
	if e := os.Getenv("AT3AM_LOG_LEVEL"); e != "" {
		logLevelStr = e
	}
	logFile, _ := cmd.Flags().GetString("log-file")
	if e := os.Getenv("AT3AM_LOG_FILE"); e != "" {
		logFile = e
	}
	challengeTypeStr := os.Getenv("AT3AM_CHALLENGE_TYPE")
	if challengeTypeStr == "" {
		challengeTypeStr = "dns-01"
	}

	// 3. Init logger
	logLevel, err := log.ParseLevel(logLevelStr)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	teardown, err := log.Init(logLevel, logFile)
	if err != nil {
		return fmt.Errorf("log init: %w", err)
	}
	defer teardown()

	// 4. DNS provider operations (record create + early-access test)
	if !skipDNS {
		// Auto-detect provider from NS records if not explicitly set
		if providerName == "" {
			detected, err := provider.Autodetect(ctx, challengeFQDN)
			if err != nil {
				return fmt.Errorf("provider autodetect failed: %w", err)
			}
			if detected == "" {
				return fmt.Errorf("could not autodetect DNS provider for %q; set AT3AM_DNS_PROVIDER or use --provider", domain)
			}
			providerName = detected
		}
		if credsPath == "" {
			credsPath = "/etc/at3am/" + providerName + ".yaml"
		}

		// Create credentials template on first run and ask user to fill it in
		created, err := provider.EnsureTemplate(credsPath, providerName)
		if err != nil {
			return fmt.Errorf("credentials template: %w", err)
		}
		if created {
			fmt.Fprintf(os.Stderr, "\nCreated credentials template: %s\n", credsPath)
			fmt.Fprintf(os.Stderr, "Fill in your %s credentials and re-run.\n\n", providerName)
			return fmt.Errorf("credentials file needs to be configured: %s", credsPath)
		}

		_, creds, err := provider.LoadCredentials(credsPath)
		if err != nil {
			return fmt.Errorf("credentials: %w", err)
		}
		p, err := provider.Lookup(ctx, providerName, creds)
		if err != nil {
			return fmt.Errorf("provider setup: %w", err)
		}
		zone, err := provider.DiscoverZone(ctx, challengeFQDN)
		if err != nil {
			return fmt.Errorf("zone discovery: %w", err)
		}

		// Early-access test: create + delete a canary record to verify credentials
		if err := provider.EarlyAccessTest(ctx, p, zone); err != nil {
			return fmt.Errorf("credential test failed (wrong credentials?): %w", err)
		}

		// Create the _acme-challenge TXT record
		relName := provider.RelativeName(challengeFQDN+".", zone)
		rec := libdns.TXT{Name: relName, TTL: 60 * time.Second, Text: validation}
		if _, err := p.AppendRecords(ctx, zone, []libdns.Record{rec}); err != nil {
			return fmt.Errorf("create TXT record: %w", err)
		}
		fmt.Printf("Created _acme-challenge TXT record for %s\n", domain)
	}

	// 5. Build config and run the propagation wait engine
	cfg := config.DefaultConfig()
	cfg.Domain = challengeFQDN
	cfg.Expected = validation
	cfg.OutputFormat = outputFormat
	cfg.ChallengeType = config.ChallengeType(challengeTypeStr)
	if err := cfg.ApplyProfile(config.Profile(profileStr)); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}

	querier := resolver.New(5 * time.Second)
	pool := resolver.NewPool(querier, nil)
	formatter := output.NewFormatter(outputFormat, os.Stdout)
	runner := wait.NewRunner(cfg, pool, formatter)
	os.Exit(runner.Run(ctx))
	return nil
}

func runManualCleanup(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// 1. Required Certbot env vars
	domain := os.Getenv("CERTBOT_DOMAIN")
	validation := os.Getenv("CERTBOT_VALIDATION")
	if domain == "" || validation == "" {
		return fmt.Errorf("CERTBOT_DOMAIN and CERTBOT_VALIDATION environment variables are required")
	}
	challengeFQDN := "_acme-challenge." + domain

	// 2. Config from flags + env vars
	logLevelStr, _ := cmd.Flags().GetString("log-level")
	if e := os.Getenv("AT3AM_LOG_LEVEL"); e != "" {
		logLevelStr = e
	}
	logFile, _ := cmd.Flags().GetString("log-file")
	if e := os.Getenv("AT3AM_LOG_FILE"); e != "" {
		logFile = e
	}
	skipDNS := os.Getenv("AT3AM_SKIP_DNS") == "1"

	// 3. Init logger
	logLevel, err := log.ParseLevel(logLevelStr)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	teardown, err := log.Init(logLevel, logFile)
	if err != nil {
		return fmt.Errorf("log init: %w", err)
	}
	defer teardown()

	// 4. Skip DNS operations if requested
	if skipDNS {
		fmt.Printf("AT3AM_SKIP_DNS=1: skipping TXT record deletion for %s\n", domain)
		return nil
	}

	// 5. Resolve provider
	providerName, _ := cmd.Flags().GetString("provider")
	if providerName == "" {
		providerName = os.Getenv("AT3AM_DNS_PROVIDER")
	}
	if providerName == "" {
		detected, err := provider.Autodetect(ctx, challengeFQDN)
		if err != nil {
			return fmt.Errorf("provider autodetect failed: %w", err)
		}
		if detected == "" {
			return fmt.Errorf("could not autodetect DNS provider for %q; set AT3AM_DNS_PROVIDER or use --provider", domain)
		}
		providerName = detected
	}

	credsPath, _ := cmd.Flags().GetString("creds")
	if credsPath == "" {
		credsPath = os.Getenv("AT3AM_DNS_CREDS")
	}
	if credsPath == "" {
		credsPath = "/etc/at3am/" + providerName + ".yaml"
	}

	_, creds, err := provider.LoadCredentials(credsPath)
	if err != nil {
		return fmt.Errorf("credentials: %w", err)
	}
	p, err := provider.Lookup(ctx, providerName, creds)
	if err != nil {
		return fmt.Errorf("provider setup: %w", err)
	}
	zone, err := provider.DiscoverZone(ctx, challengeFQDN)
	if err != nil {
		return fmt.Errorf("zone discovery: %w", err)
	}

	// 6. Delete the _acme-challenge TXT record
	// Match by name + value; the provider will handle ID lookup internally
	relName := provider.RelativeName(challengeFQDN+".", zone)
	rec := libdns.TXT{Name: relName, Text: validation}
	if _, err := p.DeleteRecords(ctx, zone, []libdns.Record{rec}); err != nil {
		// Log a warning but don't fail — cleanup is best-effort
		fmt.Fprintf(os.Stderr, "Warning: could not delete TXT record: %v\n", err)
	} else {
		fmt.Printf("Deleted _acme-challenge TXT record for %s\n", domain)
	}
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

