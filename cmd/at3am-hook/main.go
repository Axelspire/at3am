package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	Short: "ACME client manual auth hook",
	Long:  "Create DNS record and wait for propagation. Called by ACME client during validation.",
	RunE:  runManualAuth,
}

var manualCleanupCmd = &cobra.Command{
	Use:   "manual-cleanup",
	Short: "ACME client manual cleanup hook",
	Long:  "Delete DNS record after validation. Called by ACME client after validation completes.",
	RunE:  runManualCleanup,
}

var legoCmd = &cobra.Command{
	Use:   "lego",
	Short: "lego ACME client DNS hook",
	Long: `DNS hook for the lego ACME client (https://go-acme.github.io/lego).

lego sets the following environment variables before calling the hook:
  LEGO_ACTION   — "present" (create record) or "cleanup" (delete record)
  LEGO_DOMAIN   — bare domain name, e.g. example.com
  LEGO_KEY_AUTH — DNS-01 TXT record value

Configure lego to use at3am-hook:
  EXEC_PATH="at3am-hook lego" lego --dns exec ...`,
	RunE: runLego,
}

func defaultCredsPath(providerName string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}
	return filepath.Join(home, ".at3am", providerName+".yaml")
}

func init() {
	rootCmd.AddCommand(versionCmd, manualAuthCmd, manualCleanupCmd, legoCmd)

	// Flags for manual-auth
	manualAuthCmd.Flags().String("domain", "", "Domain to issue certificate for (overrides CERTBOT_DOMAIN)")
	manualAuthCmd.Flags().String("validation", "", "DNS-01 TXT record value (overrides CERTBOT_VALIDATION)")
	manualAuthCmd.Flags().String("provider", "", "DNS provider (auto-detected if not set)")
	manualAuthCmd.Flags().String("creds", "", "Path to credentials YAML file (default: ~/.at3am/<provider>.yaml)")
	manualAuthCmd.Flags().String("profile", "default", "Propagation profile: yolo, fast, default, or strict")
	manualAuthCmd.Flags().String("log-level", "warn", "Log level: debug, info, warn, error")
	manualAuthCmd.Flags().String("log-file", "", "Log file path")
	manualAuthCmd.Flags().String("output", "quiet", "Output format: quiet, text, json")
	manualAuthCmd.Flags().String("challenge-type", "dns-01", "ACME challenge type: dns-01 or persist")
	manualAuthCmd.Flags().Bool("skip-dns", false, "Skip DNS record creation/deletion (propagation wait only)")

	// Flags for manual-cleanup
	manualCleanupCmd.Flags().String("domain", "", "Domain to clean up (overrides CERTBOT_DOMAIN)")
	manualCleanupCmd.Flags().String("validation", "", "DNS-01 TXT record value to delete (overrides CERTBOT_VALIDATION)")
	manualCleanupCmd.Flags().String("provider", "", "DNS provider (auto-detected if not set)")
	manualCleanupCmd.Flags().String("creds", "", "Path to credentials YAML file (default: ~/.at3am/<provider>.yaml)")
	manualCleanupCmd.Flags().String("profile", "default", "Propagation profile: yolo, fast, default, or strict")
	manualCleanupCmd.Flags().String("log-level", "warn", "Log level: debug, info, warn, error")
	manualCleanupCmd.Flags().String("log-file", "", "Log file path")
	manualCleanupCmd.Flags().Bool("skip-dns", false, "Skip DNS record deletion")

	// Flags for lego
	legoCmd.Flags().String("action", "", "Override LEGO_ACTION: present or cleanup")
	legoCmd.Flags().String("provider", "", "DNS provider (auto-detected if not set)")
	legoCmd.Flags().String("creds", "", "Path to credentials YAML file (default: ~/.at3am/<provider>.yaml)")
	legoCmd.Flags().String("profile", "default", "Propagation profile: yolo, fast, default, or strict")
	legoCmd.Flags().String("log-level", "warn", "Log level: debug, info, warn, error")
	legoCmd.Flags().String("log-file", "", "Log file path")
	legoCmd.Flags().String("output", "quiet", "Output format: quiet, text, json")
	legoCmd.Flags().String("challenge-type", "dns-01", "ACME challenge type: dns-01 or persist")
	legoCmd.Flags().Bool("skip-dns", false, "Skip DNS record creation/deletion (propagation wait only)")
}

func runManualAuth(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// 1. Domain and validation token: flags take precedence over env vars.
	// --domain is the full challenge FQDN (e.g. _acme-challenge.example.com).
	// CERTBOT_DOMAIN is the bare domain (e.g. example.com); prefix is added below.
	domainFlag, _ := cmd.Flags().GetString("domain")
	var challengeFQDN string
	if domainFlag != "" {
		challengeFQDN = domainFlag
	} else if env := os.Getenv("CERTBOT_DOMAIN"); env != "" {
		challengeFQDN = "_acme-challenge." + env
	}
	validation, _ := cmd.Flags().GetString("validation")
	if validation == "" {
		validation = os.Getenv("CERTBOT_VALIDATION")
	}
	if challengeFQDN == "" || validation == "" {
		return fmt.Errorf("domain and validation token are required: use --domain/--validation or set CERTBOT_DOMAIN/CERTBOT_VALIDATION")
	}

	// 2. Resolve config from flags + env vars (flags take precedence over env)
	providerName, _ := cmd.Flags().GetString("provider")
	if providerName == "" {
		providerName = os.Getenv("AT3AM_DNS_PROVIDER")
	}
	credsPath, _ := cmd.Flags().GetString("creds")
	if credsPath == "" {
		credsPath = os.Getenv("AT3AM_DNS_CREDS")
	}
	skipDNSFlag, _ := cmd.Flags().GetBool("skip-dns")
	skipDNS := skipDNSFlag || os.Getenv("AT3AM_SKIP_DNS") == "1"
	profileStr, _ := cmd.Flags().GetString("profile")
	if e := os.Getenv("AT3AM_PROFILE"); e != "" {
		profileStr = e
	}
	outputFormat, _ := cmd.Flags().GetString("output")
	if e := os.Getenv("AT3AM_OUTPUT"); e != "" {
		outputFormat = e
	}
	logLevelStr, _ := cmd.Flags().GetString("log-level")
	if e := os.Getenv("AT3AM_LOG_LEVEL"); e != "" {
		logLevelStr = e
	}
	logFile, _ := cmd.Flags().GetString("log-file")
	if e := os.Getenv("AT3AM_LOG_FILE"); e != "" {
		logFile = e
	}
	challengeTypeStr, _ := cmd.Flags().GetString("challenge-type")
	if e := os.Getenv("AT3AM_CHALLENGE_TYPE"); e != "" {
		challengeTypeStr = e
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
				return fmt.Errorf("could not autodetect DNS provider for %q; set AT3AM_DNS_PROVIDER or use --provider", challengeFQDN)
			}
			providerName = detected
		}
		if credsPath == "" {
			credsPath = defaultCredsPath(providerName)
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
		fmt.Printf("Created TXT record for %s\n", challengeFQDN)
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

	querier := resolver.New(2 * time.Second)
	pool := resolver.NewPool(querier, nil)

	// Discover authoritative NS resolvers for the challenge domain so the
	// confidence engine can enforce the "ALL auth NS must confirm" requirement.
	// If discovery fails, the pool falls back to public-only (logged as a warning).
	if err := pool.DiscoverAuthNS(ctx, challengeFQDN); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: auth NS discovery failed (%v) — falling back to public resolvers only\n", err)
	}

	formatter := output.NewFormatter(outputFormat, os.Stdout)
	runner := wait.NewRunner(cfg, pool, formatter)
	os.Exit(runner.Run(ctx))
	return nil
}

func runManualCleanup(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// 1. Domain and validation token: flags take precedence over env vars.
	// --domain is the full challenge FQDN (e.g. _acme-challenge.example.com).
	// CERTBOT_DOMAIN is the bare domain (e.g. example.com); prefix is added below.
	domainFlag, _ := cmd.Flags().GetString("domain")
	var challengeFQDN string
	if domainFlag != "" {
		challengeFQDN = domainFlag
	} else if env := os.Getenv("CERTBOT_DOMAIN"); env != "" {
		challengeFQDN = "_acme-challenge." + env
	}
	validation, _ := cmd.Flags().GetString("validation")
	if validation == "" {
		validation = os.Getenv("CERTBOT_VALIDATION")
	}
	if challengeFQDN == "" || validation == "" {
		return fmt.Errorf("domain and validation token are required: use --domain/--validation or set CERTBOT_DOMAIN/CERTBOT_VALIDATION")
	}

	// 2. Config from flags + env vars
	logLevelStr, _ := cmd.Flags().GetString("log-level")
	if e := os.Getenv("AT3AM_LOG_LEVEL"); e != "" {
		logLevelStr = e
	}
	logFile, _ := cmd.Flags().GetString("log-file")
	if e := os.Getenv("AT3AM_LOG_FILE"); e != "" {
		logFile = e
	}
	skipDNSFlag, _ := cmd.Flags().GetBool("skip-dns")
	skipDNS := skipDNSFlag || os.Getenv("AT3AM_SKIP_DNS") == "1"

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
		fmt.Printf("AT3AM_SKIP_DNS=1: skipping TXT record deletion for %s\n", challengeFQDN)
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
			return fmt.Errorf("could not autodetect DNS provider for %q; set AT3AM_DNS_PROVIDER or use --provider", challengeFQDN)
		}
		providerName = detected
	}

	credsPath, _ := cmd.Flags().GetString("creds")
	if credsPath == "" {
		credsPath = os.Getenv("AT3AM_DNS_CREDS")
	}
	if credsPath == "" {
		credsPath = defaultCredsPath(providerName)
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
		fmt.Printf("Deleted TXT record for %s\n", challengeFQDN)
	}
	return nil
}

func runLego(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// 1. Action: flag overrides LEGO_ACTION env var
	action, _ := cmd.Flags().GetString("action")
	if action == "" {
		action = os.Getenv("LEGO_ACTION")
	}
	if action != "present" && action != "cleanup" {
		return fmt.Errorf("LEGO_ACTION must be 'present' or 'cleanup' (got %q); set env var or use --action", action)
	}

	// 2. Domain and TXT value from lego env vars (bare domain — prefix added below)
	legoDomain := os.Getenv("LEGO_DOMAIN")
	if legoDomain == "" {
		return fmt.Errorf("LEGO_DOMAIN is not set")
	}
	validation := os.Getenv("LEGO_KEY_AUTH")
	if validation == "" {
		return fmt.Errorf("LEGO_KEY_AUTH is not set")
	}
	challengeFQDN := "_acme-challenge." + legoDomain

	// 3. Resolve shared config from flags + env vars
	providerName, _ := cmd.Flags().GetString("provider")
	if providerName == "" {
		providerName = os.Getenv("AT3AM_DNS_PROVIDER")
	}
	credsPath, _ := cmd.Flags().GetString("creds")
	if credsPath == "" {
		credsPath = os.Getenv("AT3AM_DNS_CREDS")
	}
	skipDNSFlag, _ := cmd.Flags().GetBool("skip-dns")
	skipDNS := skipDNSFlag || os.Getenv("AT3AM_SKIP_DNS") == "1"
	profileStr, _ := cmd.Flags().GetString("profile")
	if e := os.Getenv("AT3AM_PROFILE"); e != "" {
		profileStr = e
	}
	outputFormat, _ := cmd.Flags().GetString("output")
	if e := os.Getenv("AT3AM_OUTPUT"); e != "" {
		outputFormat = e
	}
	logLevelStr, _ := cmd.Flags().GetString("log-level")
	if e := os.Getenv("AT3AM_LOG_LEVEL"); e != "" {
		logLevelStr = e
	}
	logFile, _ := cmd.Flags().GetString("log-file")
	if e := os.Getenv("AT3AM_LOG_FILE"); e != "" {
		logFile = e
	}
	challengeTypeStr, _ := cmd.Flags().GetString("challenge-type")
	if e := os.Getenv("AT3AM_CHALLENGE_TYPE"); e != "" {
		challengeTypeStr = e
	}

	// 4. Init logger
	logLevel, err := log.ParseLevel(logLevelStr)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	teardown, err := log.Init(logLevel, logFile)
	if err != nil {
		return fmt.Errorf("log init: %w", err)
	}
	defer teardown()

	// 5. Resolve provider + credentials (shared by both actions)
	var p provider.DNSProvider
	var zone string
	if !skipDNS {
		if providerName == "" {
			detected, err := provider.Autodetect(ctx, challengeFQDN)
			if err != nil {
				return fmt.Errorf("provider autodetect failed: %w", err)
			}
			if detected == "" {
				return fmt.Errorf("could not autodetect DNS provider for %q; set AT3AM_DNS_PROVIDER or use --provider", challengeFQDN)
			}
			providerName = detected
		}
		if credsPath == "" {
			credsPath = defaultCredsPath(providerName)
		}
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
		p, err = provider.Lookup(ctx, providerName, creds)
		if err != nil {
			return fmt.Errorf("provider setup: %w", err)
		}
		zone, err = provider.DiscoverZone(ctx, challengeFQDN)
		if err != nil {
			return fmt.Errorf("zone discovery: %w", err)
		}
	}

	// 6. Dispatch on action
	switch action {
	case "present":
		if !skipDNS {
			if err := provider.EarlyAccessTest(ctx, p, zone); err != nil {
				return fmt.Errorf("credential test failed (wrong credentials?): %w", err)
			}
			relName := provider.RelativeName(challengeFQDN+".", zone)
			rec := libdns.TXT{Name: relName, TTL: 60 * time.Second, Text: validation}
			if _, err := p.AppendRecords(ctx, zone, []libdns.Record{rec}); err != nil {
				return fmt.Errorf("create TXT record: %w", err)
			}
			fmt.Printf("Created TXT record for %s\n", challengeFQDN)
		}
		cfg := config.DefaultConfig()
		cfg.Domain = challengeFQDN
		cfg.Expected = validation
		cfg.OutputFormat = outputFormat
		cfg.ChallengeType = config.ChallengeType(challengeTypeStr)
		if err := cfg.ApplyProfile(config.Profile(profileStr)); err != nil {
			return fmt.Errorf("invalid profile: %w", err)
		}
		querier := resolver.New(2 * time.Second)
		pool := resolver.NewPool(querier, nil)
		if err := pool.DiscoverAuthNS(ctx, challengeFQDN); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auth NS discovery failed (%v) — falling back to public resolvers only\n", err)
		}
		formatter := output.NewFormatter(outputFormat, os.Stdout)
		runner := wait.NewRunner(cfg, pool, formatter)
		os.Exit(runner.Run(ctx))

	case "cleanup":
		if skipDNS {
			fmt.Printf("AT3AM_SKIP_DNS=1: skipping TXT record deletion for %s\n", challengeFQDN)
			return nil
		}
		relName := provider.RelativeName(challengeFQDN+".", zone)
		rec := libdns.TXT{Name: relName, Text: validation}
		if _, err := p.DeleteRecords(ctx, zone, []libdns.Record{rec}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not delete TXT record: %v\n", err)
		} else {
			fmt.Printf("Deleted TXT record for %s\n", challengeFQDN)
		}
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

