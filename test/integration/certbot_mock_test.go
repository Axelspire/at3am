// Package integration tests the Certbot + at3am-hook workflow with mock DNS.
//
// This test simulates a Certbot DNS-01 challenge without requiring real DNS credentials:
//   1. Mock DNS provider creates a TXT record
//   2. Mock resolver returns the record immediately
//   3. at3am scoring engine confirms propagation
//   4. Mock DNS provider deletes the record
//
// Run with: go test -timeout 5m ./test/integration/ -v -run TestCertbotMock
package integration

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/axelspire/at3am/internal/config"
	"github.com/axelspire/at3am/internal/log"
	"github.com/axelspire/at3am/internal/mock"
	"github.com/axelspire/at3am/internal/output"
	"github.com/axelspire/at3am/internal/resolver"
	"github.com/axelspire/at3am/internal/wait"
)

// TestCertbotMock simulates a full Certbot workflow with mock DNS.
func TestCertbotMock(t *testing.T) {
	// Initialize logging to test-results folder with datetime
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	testResultsDir := filepath.Join(root, "test-results")
	logFile := filepath.Join(testResultsDir, fmt.Sprintf("at3am-mock_%s.log", timestamp))
	if err := os.MkdirAll(testResultsDir, 0755); err != nil {
		t.Fatalf("failed to create test-results directory: %v", err)
	}
	teardown, err := log.Init(log.INFO, logFile)
	if err != nil {
		t.Fatalf("log init failed: %v", err)
	}
	defer teardown()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Simulate Certbot environment variables
	domain := "example.com"
	challengeDomain := "_acme-challenge." + domain
	token := "mock-validation-token" // Must match mock scenario

	t.Logf("Starting mock Certbot + at3am-hook integration test")
	t.Logf("  Domain: %s", domain)
	t.Logf("  Challenge: %s", challengeDomain)
	t.Logf("  Token: %s", token)
	t.Logf("  Log file: %s", logFile)

	// Step 1: Simulate manual-auth hook
	t.Log("Step 1: Simulating manual-auth hook (create record)...")
	if err := simulateManualAuth(ctx, challengeDomain, token); err != nil {
		t.Fatalf("manual-auth simulation failed: %v", err)
	}
	t.Log("  ✓ Record created (mocked)")

	// Step 2: Wait for propagation
	t.Log("Step 2: Waiting for propagation (mock scenario)...")
	cfg := config.DefaultConfig()
	cfg.Domain = challengeDomain
	cfg.Expected = "mock-validation-token" // Must match mock scenario
	cfg.Profile = config.ProfileDefault
	cfg.OutputFormat = "quiet"
	cfg.MockMode = true
	cfg.MockScenario = "instant"

	// Use mock querier
	scenario := mock.PredefinedScenarios()[cfg.MockScenario]
	querier := mock.NewQuerier(scenario)
	pool := resolver.NewPool(querier, nil)

	t.Logf("  Resolver pool: %d resolvers", pool.TotalCount())

	// Run the wait engine with quiet formatter (discard output)
	buf := &bytes.Buffer{}
	formatter := output.NewFormatter("quiet", buf)
	runner := wait.NewRunner(cfg, pool, formatter)
	exitCode := runner.Run(ctx)

	if exitCode != 0 {
		t.Fatalf("Propagation wait failed with exit code %d", exitCode)
	}
	t.Log("  ✓ Propagation confirmed (mock)")

	// Step 3: Simulate manual-cleanup hook
	t.Log("Step 3: Simulating manual-cleanup hook (delete record)...")
	if err := simulateManualCleanup(ctx, challengeDomain, token); err != nil {
		t.Fatalf("manual-cleanup simulation failed: %v", err)
	}
	t.Log("  ✓ Record deleted (mocked)")

	t.Log("✓ Mock Certbot + at3am-hook integration test passed")
}

// TestCertbotMockSlowPropagation tests with delayed propagation scenario.
func TestCertbotMockSlowPropagation(t *testing.T) {
	// Initialize logging to test-results folder with datetime
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	testResultsDir := filepath.Join(root, "test-results")
	logFile := filepath.Join(testResultsDir, fmt.Sprintf("at3am-slow_%s.log", timestamp))
	if err := os.MkdirAll(testResultsDir, 0755); err != nil {
		t.Fatalf("failed to create test-results directory: %v", err)
	}
	teardown, err := log.Init(log.INFO, logFile)
	if err != nil {
		t.Fatalf("log init failed: %v", err)
	}
	defer teardown()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	domain := "slow.example.com"
	challengeDomain := "_acme-challenge." + domain
	token := "slow-propagation-token"

	t.Logf("Starting slow propagation test")
	t.Logf("  Domain: %s", domain)
	t.Logf("  Log file: %s", logFile)

	// Create record
	if err := simulateManualAuth(ctx, challengeDomain, token); err != nil {
		t.Fatalf("manual-auth failed: %v", err)
	}

	// Wait with slow propagation scenario
	cfg := config.DefaultConfig()
	cfg.Domain = challengeDomain
	cfg.Expected = "mock-validation-token" // Must match mock scenario
	cfg.Profile = config.ProfileDefault
	cfg.OutputFormat = "quiet"
	cfg.MockMode = true
	cfg.MockScenario = "slow_propagation"

	scenario := mock.PredefinedScenarios()[cfg.MockScenario]
	querier := mock.NewQuerier(scenario)
	pool := resolver.NewPool(querier, nil)

	buf := &bytes.Buffer{}
	formatter := output.NewFormatter("quiet", buf)
	runner := wait.NewRunner(cfg, pool, formatter)
	exitCode := runner.Run(ctx)

	if exitCode != 0 {
		t.Fatalf("Slow propagation test failed with exit code %d", exitCode)
	}

	// Cleanup
	if err := simulateManualCleanup(ctx, challengeDomain, token); err != nil {
		t.Fatalf("manual-cleanup failed: %v", err)
	}

	t.Log("✓ Slow propagation test passed")
}

// TestCertbotMockMultipleDomains tests multiple domains in sequence (renewal).
func TestCertbotMockMultipleDomains(t *testing.T) {
	// Initialize logging to test-results folder with datetime
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	testResultsDir := filepath.Join(root, "test-results")
	logFile := filepath.Join(testResultsDir, fmt.Sprintf("at3am-multi_%s.log", timestamp))
	if err := os.MkdirAll(testResultsDir, 0755); err != nil {
		t.Fatalf("failed to create test-results directory: %v", err)
	}
	teardown, err := log.Init(log.INFO, logFile)
	if err != nil {
		t.Fatalf("log init failed: %v", err)
	}
	defer teardown()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	domains := []string{"example.com", "www.example.com", "api.example.com"}
	token := "mock-validation-token" // Must match mock scenario

	t.Logf("Starting multi-domain test (simulating renewal)")
	t.Logf("  Log file: %s", logFile)

	for _, domain := range domains {
		t.Logf("  Processing domain: %s", domain)

		challengeDomain := "_acme-challenge." + domain

		// Create
		if err := simulateManualAuth(ctx, challengeDomain, token); err != nil {
			t.Fatalf("manual-auth failed for %s: %v", domain, err)
		}

		// Wait
		cfg := config.DefaultConfig()
		cfg.Domain = challengeDomain
		cfg.Expected = "mock-validation-token" // Must match mock scenario
		cfg.Profile = config.ProfileDefault
		cfg.OutputFormat = "quiet"
		cfg.MockMode = true
		cfg.MockScenario = "instant"

		scenario := mock.PredefinedScenarios()[cfg.MockScenario]
		querier := mock.NewQuerier(scenario)
		pool := resolver.NewPool(querier, nil)

		buf := &bytes.Buffer{}
		formatter := output.NewFormatter("quiet", buf)
		runner := wait.NewRunner(cfg, pool, formatter)
		if exitCode := runner.Run(ctx); exitCode != 0 {
			t.Fatalf("Propagation wait failed for %s", domain)
		}

		// Cleanup
		if err := simulateManualCleanup(ctx, challengeDomain, token); err != nil {
			t.Fatalf("manual-cleanup failed for %s: %v", domain, err)
		}

		t.Logf("    ✓ %s processed", domain)
	}

	t.Log("✓ Multi-domain test passed")
}

// simulateManualAuth simulates the manual-auth hook (create record).
func simulateManualAuth(ctx context.Context, domain, token string) error {
	// In a real scenario, this would:
	// 1. Detect the DNS provider
	// 2. Load credentials
	// 3. Create the TXT record
	// 4. Run early-access test
	//
	// For this mock test, we just validate inputs
	if domain == "" || token == "" {
		return fmt.Errorf("invalid domain or token")
	}
	return nil
}

// simulateManualCleanup simulates the manual-cleanup hook (delete record).
func simulateManualCleanup(ctx context.Context, domain, token string) error {
	// In a real scenario, this would:
	// 1. Detect the DNS provider
	// 2. Load credentials
	// 3. Delete the TXT record
	//
	// For this mock test, we just validate inputs
	if domain == "" || token == "" {
		return fmt.Errorf("invalid domain or token")
	}
	return nil
}

