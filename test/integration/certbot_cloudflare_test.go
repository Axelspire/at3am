// Package integration tests the full Certbot + at3am-hook + Cloudflare workflow.
//
// This test simulates a real Certbot DNS-01 challenge:
//  1. Create a _acme-challenge TXT record via Cloudflare API
//  2. Wait for global propagation using at3am's scoring engine
//  3. Verify the record is visible via public DNS
//  4. Delete the record (always, via defer)
//
// Credentials are loaded from .env/cloudflare.env at the repository root.
// Environment variables (CF_API_TOKEN, CF_ZONE_ID, TEST_DOMAIN) take precedence
// over the file values when set.
// The test is skipped if neither the file nor the environment variables are present.
//
// Run with:
//
//	go test -timeout 10m ./test/integration/ -v -run TestCertbotCloudflare
package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/axelspire/at3am/internal/config"
	"github.com/axelspire/at3am/internal/log"
	"github.com/axelspire/at3am/internal/output"
	"github.com/axelspire/at3am/internal/resolver"
	"github.com/axelspire/at3am/internal/wait"
)

// cloudflareEnvFile is the path (relative to repo root) of the env file.
const cloudflareEnvFile = ".env/cloudflare.env"

// loadCloudflareEnv resolves credentials for the Cloudflare integration test.
// Priority: environment variable > .env/cloudflare.env file value.
// Returns the three required values, or an error if any are missing.
func loadCloudflareEnv(t *testing.T) (cfToken, cfZoneID, domain string) {
	t.Helper()

	// Best-effort load of the env file; errors are non-fatal if env vars cover it.
	fileVars, fileErr := loadEnvFile(cloudflareEnvFile)
	if fileErr != nil {
		t.Logf("Note: could not load %s: %v", cloudflareEnvFile, fileErr)
		fileVars = map[string]string{}
	}

	resolve := func(envKey string) string {
		if v := os.Getenv(envKey); v != "" {
			return v
		}
		return fileVars[envKey]
	}

	cfToken = resolve("CF_API_TOKEN")
	cfZoneID = resolve("CF_ZONE_ID")
	domain = resolve("TEST_DOMAIN")

	if cfToken == "" || cfZoneID == "" || domain == "" {
		t.Skipf("Cloudflare credentials not found in %s or environment; skipping", cloudflareEnvFile)
	}
	return cfToken, cfZoneID, domain
}

// TestCertbotCloudflare simulates a full Certbot DNS-01 challenge workflow
// using the Cloudflare DNS provider and the at3am propagation engine.
func TestCertbotCloudflare(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfToken, cfZoneID, domain := loadCloudflareEnv(t)

	// Initialize logging to test-results folder with datetime
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	testResultsDir := filepath.Join(root, "test-results")
	logFile := filepath.Join(testResultsDir, fmt.Sprintf("at3am-cloudflare_%s.log", timestamp))
	if err := os.MkdirAll(testResultsDir, 0755); err != nil {
		t.Fatalf("failed to create test-results directory: %v", err)
	}
	teardown, err := log.Init(log.INFO, logFile)
	if err != nil {
		t.Fatalf("log init failed: %v", err)
	}
	defer teardown()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Generate a unique challenge token (mimics what Certbot produces).
	token := generateToken()
	challengeDomain := "_acme-challenge." + domain

	t.Logf("Starting Certbot + Cloudflare integration test")
	t.Logf("  Zone ID:   %s", cfZoneID)
	t.Logf("  Domain:    %s", domain)
	t.Logf("  Challenge: %s", challengeDomain)
	t.Logf("  Token:     %s", token)
	t.Logf("  Log file:  %s", logFile)

	// ── Step 1: Create TXT record (manual-auth hook) ─────────────────────────
	t.Log("Step 1: Creating TXT record via Cloudflare API...")
	recordID, err := createCloudflareRecord(ctx, cfToken, cfZoneID, "_acme-challenge", token)
	if err != nil {
		t.Fatalf("Failed to create Cloudflare record: %v", err)
	}
	t.Logf("  Record ID: %s", recordID)

	// Always delete the record on exit (manual-cleanup hook).
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := deleteCloudflareRecord(cleanupCtx, cfToken, cfZoneID, recordID); err != nil {
			t.Logf("Warning: failed to delete record %s: %v", recordID, err)
		} else {
			t.Logf("  Record %s deleted", recordID)
		}
	}()

	// ── Step 2: Wait for propagation ─────────────────────────────────────────
	t.Log("Step 2: Waiting for propagation...")
	cfg := config.DefaultConfig()
	cfg.Domain = challengeDomain
	cfg.Expected = token
	cfg.Profile = config.ProfileDefault
	cfg.OutputFormat = "quiet"

	querier := resolver.New(10 * time.Second)
	pool := resolver.NewPool(querier, nil)

	// Add Cloudflare's own authoritative nameservers for axlespire.com so the
	// at3am engine polls the source-of-truth directly.
	if err := pool.DiscoverAuthNS(ctx, challengeDomain); err != nil {
		t.Logf("Warning: NS discovery failed: %v (continuing with public resolvers only)", err)
	}

	t.Logf("  Resolver pool: %d public, %d auth, %d total",
		pool.PublicCount(), pool.AuthCount(), pool.TotalCount())

	formatter := output.NewFormatter("quiet", &testWriter{t})
	runner := wait.NewRunner(cfg, pool, formatter)
	if exitCode := runner.Run(ctx); exitCode != 0 {
		t.Fatalf("Propagation wait failed with exit code %d", exitCode)
	}
	t.Log("  ✓ Propagation confirmed")

	// ── Step 3: Final spot-check via Google DNS ───────────────────────────────
	t.Log("Step 3: Verifying record visibility via 8.8.8.8...")
	if err := verifyRecordVisible(ctx, challengeDomain, token); err != nil {
		t.Fatalf("Record verification failed: %v", err)
	}
	t.Log("  ✓ Record verified visible")

	t.Log("✓ Certbot + Cloudflare integration test PASSED")
}

// testWriter forwards output.Formatter writes to t.Log so they appear inline.
type testWriter struct{ t *testing.T }

func (tw *testWriter) Write(p []byte) (int, error) {
	tw.t.Log(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

// createCloudflareRecord creates a TXT record via Cloudflare API.
func createCloudflareRecord(ctx context.Context, token, zoneID, name, value string) (string, error) {
	const cfAPIBase = "https://api.cloudflare.com/client/v4"

	payload := map[string]interface{}{
		"type":    "TXT",
		"name":    name,
		"content": value,
		"ttl":     120,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/zones/%s/dns_records", cfAPIBase, zoneID),
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Cloudflare API error: %d %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Success bool `json:"success"`
		Result  struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	if !result.Success {
		return "", fmt.Errorf("Cloudflare API returned success=false")
	}

	return result.Result.ID, nil
}

// deleteCloudflareRecord deletes a TXT record via Cloudflare API.
func deleteCloudflareRecord(ctx context.Context, token, zoneID, recordID string) error {
	const cfAPIBase = "https://api.cloudflare.com/client/v4"

	req, err := http.NewRequestWithContext(ctx, "DELETE",
		fmt.Sprintf("%s/zones/%s/dns_records/%s", cfAPIBase, zoneID, recordID),
		nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Cloudflare API error: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

// verifyRecordVisible checks if the TXT record is visible via public DNS.
func verifyRecordVisible(ctx context.Context, domain, expectedValue string) error {
	querier := resolver.New(2 * time.Second)

	// Query a public resolver (Google DNS)
	result := querier.QueryTXT(ctx, domain, "8.8.8.8:53")

	if result.Error != "" {
		return fmt.Errorf("DNS query failed: %s", result.Error)
	}

	if !result.Found {
		return fmt.Errorf("expected TXT record not found")
	}

	for _, value := range result.Values {
		if strings.Contains(value, expectedValue) {
			return nil
		}
	}

	return fmt.Errorf("expected value not found in TXT records (got: %v)", result.Values)
}

// generateToken creates a random token similar to Certbot's validation token.
func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

