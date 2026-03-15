// Package integration runs real DNS propagation tests using Cloudflare API.
//
// Credentials are loaded from .env/cloudflare.env at the repository root.
// Environment variables (CF_API_TOKEN, CF_ZONE_ID, TEST_DOMAIN) take precedence
// over file values when set.
//
// Run with: AT3AM_INTEGRATION=1 go test -timeout 30m ./test/integration/ -v
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

	"github.com/axelspire/at3am/internal/confidence"
	"github.com/axelspire/at3am/internal/diagnostics"
	"github.com/axelspire/at3am/internal/log"
	"github.com/axelspire/at3am/internal/resolver"
	"github.com/axelspire/at3am/internal/ttl"
)

const cfAPIBase = "https://api.cloudflare.com/client/v4"

// propagationCreds holds the resolved Cloudflare credentials for a test run.
type propagationCreds struct {
	token  string
	zoneID string
	domain string
}

// loadPropagationCreds resolves credentials from .env/cloudflare.env and/or
// environment variables. The test is skipped if any value is missing.
func loadPropagationCreds(t *testing.T) propagationCreds {
	t.Helper()

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

	creds := propagationCreds{
		token:  resolve("CF_API_TOKEN"),
		zoneID: resolve("CF_ZONE_ID"),
		domain: resolve("TEST_DOMAIN"),
	}

	if creds.token == "" || creds.zoneID == "" || creds.domain == "" {
		t.Skipf("Cloudflare credentials not found in %s or environment; skipping", cloudflareEnvFile)
	}
	return creds
}

// TestResult captures a single poll measurement.
type TestResult struct {
	Timestamp      time.Time `json:"timestamp"`
	ElapsedSec     float64   `json:"elapsed_sec"`
	Attempt        int       `json:"attempt"`
	AuthScore      float64   `json:"auth_score"`
	PublicScore    float64   `json:"public_score"`
	Overall        float64   `json:"overall"`
	AuthFound      int       `json:"auth_found"`
	AuthTotal      int       `json:"auth_total"`
	AuthRequired   int       `json:"auth_required"`
	AuthErrors     int       `json:"auth_errors"`
	PublicFound    int       `json:"public_found"`
	PublicTotal    int       `json:"public_total"`
	PublicRequired int       `json:"public_required"`
	PublicErrors   int       `json:"public_errors"`
	// Per-profile readiness flags for this attempt.
	ReadyYolo    bool   `json:"ready_yolo"`
	ReadyFast    bool   `json:"ready_fast"`
	ReadyDefault bool   `json:"ready_default"`
	ReadyStrict  bool   `json:"ready_strict"`
	Scenario     string `json:"scenario"`
	TTLMin       uint32 `json:"ttl_min"`
	TTLMax       uint32 `json:"ttl_max"`
}

// PropagationTest encapsulates one full test run.
type PropagationTest struct {
	Name          string             `json:"name"`
	Token         string             `json:"token"`
	FQDN          string             `json:"fqdn"`
	TTL           int                `json:"ttl"`
	RecordID      string             `json:"record_id,omitempty"`
	RecordCreated time.Time          `json:"record_created"`
	FirstSeen     time.Time          `json:"first_seen,omitempty"`
	Results       []TestResult       `json:"results"`
	// ReadyAt records the elapsed seconds at which each profile first became ready.
	ReadyAt       map[string]float64 `json:"ready_at"`
}

func randomToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// cfRequest makes an authenticated Cloudflare API request with detailed logging.
func cfRequest(creds propagationCreds, method, path string, body interface{}) (map[string]interface{}, error) {
	var reqBody io.Reader
	var reqBodyStr string
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
		reqBodyStr = string(data)
	}

	log.Debug("[API] %s %s%s", method, cfAPIBase, path)
	if reqBodyStr != "" {
		log.Debug("[API] Request Body: %s", reqBodyStr)
	}

	req, err := http.NewRequest(method, cfAPIBase+path, reqBody)
	if err != nil {
		log.Error("[API] Request creation failed: %v", err)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+creds.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error("[API] Request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	respData, _ := io.ReadAll(resp.Body)

	log.Debug("[API] Response Status: %s", resp.Status)
	log.Debug("[API] Response Body: %s", string(respData))

	var result map[string]interface{}
	if err := json.Unmarshal(respData, &result); err != nil {
		log.Error("[API] Failed to parse response: %v", err)
		return nil, fmt.Errorf("failed to parse response: %s", string(respData))
	}
	if success, ok := result["success"].(bool); !ok || !success {
		log.Error("[API] API call failed: %s", string(respData))
		return nil, fmt.Errorf("API error: %s", string(respData))
	}

	log.Debug("[API] API call successful")
	return result, nil
}

// createTXTRecord creates a TXT record via Cloudflare API.
func createTXTRecord(creds propagationCreds, subdomain, value string, ttlSec int) (string, error) {
	fqdn := subdomain + "." + creds.domain
	quotedValue := fmt.Sprintf("\"%s\"", value)
	body := map[string]interface{}{
		"type":    "TXT",
		"name":    fqdn,
		"content": quotedValue,
		"ttl":     ttlSec,
	}
	result, err := cfRequest(creds, "POST", "/zones/"+creds.zoneID+"/dns_records", body)
	if err != nil {
		return "", err
	}
	rec := result["result"].(map[string]interface{})
	return rec["id"].(string), nil
}

// deleteTXTRecord removes a TXT record via Cloudflare API.
func deleteTXTRecord(creds propagationCreds, recordID string) error {
	_, err := cfRequest(creds, "DELETE", "/zones/"+creds.zoneID+"/dns_records/"+recordID, nil)
	return err
}

// cleanupACMERecords removes all _acme-challenge TXT records for the given prefix.
func cleanupACMERecords(creds propagationCreds, prefix string) {
	path := fmt.Sprintf("/zones/%s/dns_records?type=TXT&name=%s.%s", creds.zoneID, prefix, creds.domain)
	result, err := cfRequest(creds, "GET", path, nil)
	if err != nil {
		return
	}
	records, ok := result["result"].([]interface{})
	if !ok {
		return
	}
	for _, r := range records {
		rec := r.(map[string]interface{})
		id := rec["id"].(string)
		_ = deleteTXTRecord(creds, id)
	}
}

// runPropagationTest creates a record and polls until full propagation or timeout.
func runPropagationTest(t *testing.T, creds propagationCreds, name, subdomain string, ttlSec int, timeout time.Duration) PropagationTest {
	t.Helper()
	token := randomToken()
	fqdn := subdomain + "." + creds.domain

	pt := PropagationTest{
		Name:    name,
		Token:   token,
		FQDN:    fqdn,
		TTL:     ttlSec,
		ReadyAt: make(map[string]float64),
	}

	// Clean up any leftover records
	cleanupACMERecords(creds, subdomain)
	time.Sleep(1 * time.Second)

	// Create the record
	t.Logf("[%s] Creating TXT record: %s = %s (TTL=%d)", name, fqdn, token, ttlSec)
	recordID, err := createTXTRecord(creds, subdomain, token, ttlSec)
	if err != nil {
		t.Fatalf("failed to create TXT record: %v", err)
	}
	pt.RecordID = recordID
	pt.RecordCreated = time.Now()
	defer func() {
		t.Logf("[%s] Cleaning up record %s", name, recordID)
		_ = deleteTXTRecord(creds, recordID)
	}()

	// Set up resolver pool
	querier := resolver.New(5 * time.Second)
	pool := resolver.NewPool(querier, nil)
	if err := pool.DiscoverAuthNS(context.Background(), fqdn); err != nil {
		t.Logf("[%s] Warning: could not discover auth NS: %v", name, err)
	}
	t.Logf("[%s] Resolver pool: %d public + %d auth = %d total",
		name, pool.PublicCount(), pool.AuthCount(), pool.TotalCount())

	// One engine per profile — each implements the correct readiness thresholds.
	engines := map[string]*confidence.Engine{
		"yolo":    confidence.NewEngineWithProfile(confidence.ProfileYolo),
		"fast":    confidence.NewEngineWithProfile(confidence.ProfileFast),
		"default": confidence.NewEngineWithProfile(confidence.ProfileDefault),
		"strict":  confidence.NewEngineWithProfile(confidence.ProfileStrict),
	}
	ttlAnalyser := ttl.NewAnalyser()
	diagEngine := diagnostics.NewEngine()

	// Poll loop
	start := pt.RecordCreated
	deadline := start.Add(timeout)
	attempt := 0
	interval := 1 * time.Second

	t.Logf("[%s] Polling started (timeout=%s, interval=%s)", name, timeout, interval)
	t.Logf("[%s] %-6s %-8s %-8s %-8s %-10s %-10s %-10s %s",
		name, "Att#", "Auth%", "Public%", "Overall%", "Auth", "Public", "Errors", "Scenario")
	t.Logf("[%s] %s", name, strings.Repeat("-", 80))

	for time.Now().Before(deadline) {
		attempt++
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		results := pool.QueryAuthFirst(ctx, fqdn)
		cancel()

		// Use the default engine for display scores; each profile engine for readiness.
		score := engines["default"].Calculate(results, token)
		ttlReport := ttlAnalyser.Analyse(results)
		diag := diagEngine.Diagnose(results, score, ttlReport)
		elapsed := time.Since(start).Seconds()

		// Per-profile readiness for this attempt.
		scoreYolo    := engines["yolo"].Calculate(results, token)
		scoreFast    := engines["fast"].Calculate(results, token)
		scoreStrict  := engines["strict"].Calculate(results, token)

		tr := TestResult{
			Timestamp:      time.Now(),
			ElapsedSec:     elapsed,
			Attempt:        attempt,
			AuthScore:      score.AuthScore,
			PublicScore:    score.PublicScore,
			Overall:        score.Overall,
			AuthFound:      score.AuthFound,
			AuthTotal:      score.AuthTotal,
			AuthRequired:   score.AuthRequired,
			AuthErrors:     score.AuthErrors,
			PublicFound:    score.PublicFound,
			PublicTotal:    score.PublicTotal,
			PublicRequired: score.PublicRequired,
			PublicErrors:   score.PublicErrors,
			ReadyYolo:      scoreYolo.Ready,
			ReadyFast:      scoreFast.Ready,
			ReadyDefault:   score.Ready,
			ReadyStrict:    scoreStrict.Ready,
			Scenario:       diag.Scenario,
			TTLMin:         ttlReport.MinTTL,
			TTLMax:         ttlReport.MaxTTL,
		}
		pt.Results = append(pt.Results, tr)

		// Log the row
		t.Logf("[%s] %-6d %-8.1f %-8.1f %-8.1f %-10s %-10s %-10s %s",
			name, attempt,
			score.AuthScore, score.PublicScore, score.Overall,
			fmt.Sprintf("%d/%d (need %d)", score.AuthFound, score.AuthTotal, score.AuthRequired),
			fmt.Sprintf("%d/%d (need %d)", score.PublicFound, score.PublicTotal, score.PublicRequired),
			fmt.Sprintf("a:%d p:%d", score.AuthErrors, score.PublicErrors),
			diag.Scenario)

		// Track first seen
		if pt.FirstSeen.IsZero() && (score.AuthFound > 0 || score.PublicFound > 0) {
			pt.FirstSeen = time.Now()
			t.Logf("[%s] *** First seen at %.1fs", name, elapsed)
		}

		// Record the first elapsed time at which each profile becomes ready.
		profileReady := map[string]bool{
			"yolo":    scoreYolo.Ready,
			"fast":    scoreFast.Ready,
			"default": score.Ready,
			"strict":  scoreStrict.Ready,
		}
		for prof, ready := range profileReady {
			if _, alreadyCrossed := pt.ReadyAt[prof]; !alreadyCrossed && ready {
				pt.ReadyAt[prof] = elapsed
				t.Logf("[%s] *** Profile '%s' READY at %.1fs (auth %d/%d need %d, pub %d/%d need %d)",
					name, prof,
					elapsed,
					score.AuthFound, score.AuthTotal, score.AuthRequired,
					score.PublicFound, score.PublicTotal, score.PublicRequired)
			}
		}

		// Stop once the strictest profile we care about (default) is satisfied.
		if score.Ready {
			t.Logf("[%s] *** Default profile READY at %.1fs", name, elapsed)
			break
		}

		// Write progress file every 10 attempts
		if attempt%10 == 0 {
			writeProgressFile(t, name, pt)
		}

		time.Sleep(interval)
	}

	return pt
}

// testResultsDir returns the path to the test-results directory,
// creating it if necessary.
func testResultsDir() string {
	dir := "test-results"
	// If running from test/integration/, write to repo root's test-results/
	if _, err := os.Stat("../../go.mod"); err == nil {
		dir = "../../test-results"
	}
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// writeProgressFile writes current test progress to a file.
func writeProgressFile(t *testing.T, testName string, pt PropagationTest) {
	dir := testResultsDir()
	progressFile := fmt.Sprintf("%s/progress_%s_%s.json", dir, testName, time.Now().Format("150405"))
	data, err := json.MarshalIndent(pt, "", "  ")
	if err != nil {
		t.Logf("Warning: could not marshal progress: %v", err)
		return
	}
	if err := os.WriteFile(progressFile, data, 0644); err != nil {
		t.Logf("Warning: could not write progress file: %v", err)
		return
	}
	t.Logf("[%s] Progress written to: %s", testName, progressFile)
}

// writeDetailedLog writes full JSON results to a log file.
func writeDetailedLog(t *testing.T, tests []PropagationTest) {
	dir := testResultsDir()
	logFile := fmt.Sprintf("%s/propagation_%s.json", dir, time.Now().Format("20060102_150405"))
	data, err := json.MarshalIndent(tests, "", "  ")
	if err != nil {
		t.Logf("Warning: could not marshal log: %v", err)
		return
	}
	if err := os.WriteFile(logFile, data, 0644); err != nil {
		t.Logf("Warning: could not write log file: %v", err)
		return
	}
	t.Logf("Detailed log written to: %s", logFile)
}

// writeCSVLog writes a CSV version of results for easy analysis.
func writeCSVLog(t *testing.T, tests []PropagationTest) {
	dir := testResultsDir()
	csvFile := fmt.Sprintf("%s/propagation_%s.csv", dir, time.Now().Format("20060102_150405"))
	var buf bytes.Buffer
	buf.WriteString("test_name,attempt,elapsed_sec,auth_score,public_score,overall," +
		"auth_found,auth_total,auth_required,auth_errors," +
		"public_found,public_total,public_required,public_errors," +
		"ready_yolo,ready_fast,ready_default,ready_strict,scenario,ttl_min,ttl_max\n")
	for _, pt := range tests {
		for _, r := range pt.Results {
			buf.WriteString(fmt.Sprintf("%s,%d,%.1f,%.1f,%.1f,%.1f,%d,%d,%d,%d,%d,%d,%d,%d,%t,%t,%t,%t,%s,%d,%d\n",
				pt.Name, r.Attempt, r.ElapsedSec,
				r.AuthScore, r.PublicScore, r.Overall,
				r.AuthFound, r.AuthTotal, r.AuthRequired, r.AuthErrors,
				r.PublicFound, r.PublicTotal, r.PublicRequired, r.PublicErrors,
				r.ReadyYolo, r.ReadyFast, r.ReadyDefault, r.ReadyStrict,
				r.Scenario, r.TTLMin, r.TTLMax))
		}
	}
	if err := os.WriteFile(csvFile, buf.Bytes(), 0644); err != nil {
		t.Logf("Warning: could not write CSV file: %v", err)
		return
	}
	t.Logf("CSV log written to: %s", csvFile)
}

// printSummary outputs a formatted summary of all test results.
func printSummary(t *testing.T, tests []PropagationTest) {
	t.Helper()
	profileOrder := []string{"yolo", "fast", "default", "strict"}

	t.Logf("\n%s", strings.Repeat("=", 100))
	t.Logf("PROPAGATION TEST SUMMARY")
	t.Logf("%s", strings.Repeat("=", 100))

	for _, pt := range tests {
		t.Logf("\n--- %s ---", pt.Name)
		t.Logf("  FQDN:           %s", pt.FQDN)
		t.Logf("  TTL:            %d seconds", pt.TTL)
		t.Logf("  Token:          %s", pt.Token)
		t.Logf("  Record created: %s", pt.RecordCreated.Format(time.RFC3339))
		t.Logf("  Total polls:    %d", len(pt.Results))

		if !pt.FirstSeen.IsZero() {
			t.Logf("  First seen:     %.1fs after creation", pt.FirstSeen.Sub(pt.RecordCreated).Seconds())
		} else {
			t.Logf("  First seen:     NEVER (within timeout)")
		}

		// Per-profile readiness
		t.Logf("  Profile readiness (READY = auth_correct >= auth_threshold AND public_correct >= public_threshold):")
		t.Logf("    %-10s %-28s %-28s %s", "Profile", "Auth threshold", "Public threshold", "Ready at")
		t.Logf("    %s", strings.Repeat("-", 80))
		profileDesc := map[string][2]string{
			"yolo":    {"≥1", "≥0"},
			"fast":    {"≥ceil(N/2)", "≥1"},
			"default": {"ALL", "≥1"},
			"strict":  {"ALL", "ALL−2"},
		}
		for _, prof := range profileOrder {
			desc := profileDesc[prof]
			if elapsed, ok := pt.ReadyAt[prof]; ok {
				t.Logf("    %-10s %-28s %-28s %.1fs", prof, desc[0], desc[1], elapsed)
			} else {
				t.Logf("    %-10s %-28s %-28s NOT READY within timeout", prof, desc[0], desc[1])
			}
		}

		// Last recorded scores
		if len(pt.Results) > 0 {
			last := pt.Results[len(pt.Results)-1]
			t.Logf("  Final scores: auth=%.1f%% (%d/%d need %d) public=%.1f%% (%d/%d need %d) overall=%.1f%%",
				last.AuthScore, last.AuthFound, last.AuthTotal, last.AuthRequired,
				last.PublicScore, last.PublicFound, last.PublicTotal, last.PublicRequired,
				last.Overall)
		}
	}

	// Comparison table across all tests
	t.Logf("\n%s", strings.Repeat("=", 100))
	t.Logf("PROFILE READINESS COMPARISON ACROSS TESTS")
	t.Logf("%s", strings.Repeat("=", 100))
	t.Logf("%-30s %-12s %-12s %-12s %-12s", "Test", "yolo", "fast", "default", "strict")
	t.Logf("%s", strings.Repeat("-", 80))
	for _, pt := range tests {
		vals := make([]string, len(profileOrder))
		for i, prof := range profileOrder {
			if elapsed, ok := pt.ReadyAt[prof]; ok {
				vals[i] = fmt.Sprintf("%.1fs", elapsed)
			} else {
				vals[i] = "---"
			}
		}
		t.Logf("%-30s %-12s %-12s %-12s %-12s", pt.Name, vals[0], vals[1], vals[2], vals[3])
	}
}

// TestPropagation is the main integration test.
// Run with: AT3AM_INTEGRATION=1 go test -timeout 30m -v ./test/integration/
//
// Each run generates a unique 8-hex-char ID used as a subdomain prefix for all
// sub-tests. This ensures completely fresh FQDNs per run, bypassing any
// negative-cache (NXDOMAIN) entries from previous create/delete cycles on the
// same name.
func TestPropagation(t *testing.T) {
	if os.Getenv("AT3AM_INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set AT3AM_INTEGRATION=1 to run.")
	}

	// Initialize logging to test-results folder with datetime
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	testResultsDir := filepath.Join(root, "test-results")
	logFile := filepath.Join(testResultsDir, fmt.Sprintf("at3am-propagation_%s.log", timestamp))
	if err := os.MkdirAll(testResultsDir, 0755); err != nil {
		t.Fatalf("failed to create test-results directory: %v", err)
	}
	teardown, err := log.Init(log.DEBUG, logFile)
	if err != nil {
		t.Fatalf("log init failed: %v", err)
	}
	defer teardown()

	creds := loadPropagationCreds(t)
	t.Logf("Domain: %s  Zone: %s", creds.domain, creds.zoneID)

	// Unique 8-hex run ID — guarantees fresh FQDNs across runs.
	runID := randomToken()[:8]
	t.Logf("Run ID: %s (all FQDNs prefixed with this ID)", runID)
	t.Logf("Log file: %s", logFile)

	var allResults []PropagationTest
	var recordsToCleanup []struct {
		subdomain string
		recordID  string
	}

	// Ensure cleanup happens even if test times out
	defer func() {
		t.Logf("Cleaning up %d DNS records...", len(recordsToCleanup))
		for _, rec := range recordsToCleanup {
			if rec.recordID != "" {
				t.Logf("Deleting record %s (%s)", rec.recordID, rec.subdomain)
				_ = deleteTXTRecord(creds, rec.recordID)
			}
		}
	}()

	// Test 1: Low TTL (60s) — simulates a typical ACME challenge record.
	t.Run("LowTTL_60s", func(t *testing.T) {
		subdomain := fmt.Sprintf("_acme-%s", runID)
		pt := runPropagationTest(t, creds, "LowTTL_60s", subdomain, 60, 5*time.Minute)
		allResults = append(allResults, pt)
		recordsToCleanup = append(recordsToCleanup, struct {
			subdomain string
			recordID  string
		}{subdomain, pt.RecordID})
	})

	// Brief pause between tests to avoid hitting Cloudflare rate limits.
	time.Sleep(5 * time.Second)

	// Test 2: Auto TTL (Cloudflare default ~300s). TTL=1 means "auto" in CF API.
	t.Run("AutoTTL", func(t *testing.T) {
		subdomain := fmt.Sprintf("_acme-auto-%s", runID)
		pt := runPropagationTest(t, creds, "AutoTTL", subdomain, 1, 5*time.Minute)
		allResults = append(allResults, pt)
		recordsToCleanup = append(recordsToCleanup, struct {
			subdomain string
			recordID  string
		}{subdomain, pt.RecordID})
	})

	time.Sleep(5 * time.Second)

	// Test 3: Subdomain challenge (deeper label path).
	t.Run("SubdomainChallenge", func(t *testing.T) {
		subdomain := fmt.Sprintf("_acme-%s.www", runID)
		pt := runPropagationTest(t, creds, "SubdomainChallenge", subdomain, 60, 5*time.Minute)
		allResults = append(allResults, pt)
		recordsToCleanup = append(recordsToCleanup, struct {
			subdomain string
			recordID  string
		}{subdomain, pt.RecordID})
	})

	// Output summary and logs
	if len(allResults) > 0 {
		printSummary(t, allResults)
		writeDetailedLog(t, allResults)
		writeCSVLog(t, allResults)
	}
}
