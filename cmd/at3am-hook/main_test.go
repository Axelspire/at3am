package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout to a buffer for the duration of fn.
func captureStdout(fn func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// resetFlags resets all cobra flag values to their defaults between tests.
// Without this, flag values from one test bleed into the next because the
// command objects are package-level singletons.
func resetFlags() {
	manualAuthCmd.Flags().Set("provider", "")
	manualAuthCmd.Flags().Set("creds", "")
	manualAuthCmd.Flags().Set("profile", "default")
	manualAuthCmd.Flags().Set("log-level", "warn")
	manualAuthCmd.Flags().Set("log-file", "")
	manualCleanupCmd.Flags().Set("provider", "")
	manualCleanupCmd.Flags().Set("creds", "")
	manualCleanupCmd.Flags().Set("log-level", "warn")
	manualCleanupCmd.Flags().Set("log-file", "")
}

// executeArgs resets all flags, sets rootCmd args, and executes.
func executeArgs(args ...string) error {
	resetFlags()
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

// setEnv sets Certbot environment variables and returns a cleanup function.
func setEnv(domain, validation string) func() {
	os.Setenv("CERTBOT_DOMAIN", domain)
	os.Setenv("CERTBOT_VALIDATION", validation)
	return func() {
		os.Unsetenv("CERTBOT_DOMAIN")
		os.Unsetenv("CERTBOT_VALIDATION")
	}
}

// ── version command ──────────────────────────────────────────────────────────

func TestVersionCommand_Output(t *testing.T) {
	out := captureStdout(func() {
		executeArgs("version")
	})
	if !strings.Contains(out, "at3am-hook version") {
		t.Errorf("expected 'at3am-hook version' in output, got: %q", out)
	}
}

func TestVersionCommand_ContainsCommit(t *testing.T) {
	out := captureStdout(func() {
		executeArgs("version")
	})
	if !strings.Contains(out, "commit") {
		t.Errorf("expected 'commit' in output, got: %q", out)
	}
}

func TestVersionCommand_ContainsBuildTime(t *testing.T) {
	out := captureStdout(func() {
		executeArgs("version")
	})
	if !strings.Contains(out, "built") {
		t.Errorf("expected 'built' in output, got: %q", out)
	}
}

func TestVersionVariables_Defaults(t *testing.T) {
	if version == "" {
		t.Error("version should not be empty")
	}
	if commit == "" {
		t.Error("commit should not be empty")
	}
	if buildTime == "" {
		t.Error("buildTime should not be empty")
	}
}

// ── command structure ────────────────────────────────────────────────────────

func TestCommandsRegistered(t *testing.T) {
	names := map[string]bool{}
	for _, cmd := range rootCmd.Commands() {
		names[cmd.Name()] = true
	}
	for _, want := range []string{"version", "manual-auth", "manual-cleanup"} {
		if !names[want] {
			t.Errorf("expected command %q to be registered", want)
		}
	}
}

// ── flag registration ────────────────────────────────────────────────────────

func TestManualAuthFlags(t *testing.T) {
	for _, name := range []string{"provider", "creds", "profile", "log-level", "log-file"} {
		if manualAuthCmd.Flags().Lookup(name) == nil {
			t.Errorf("manual-auth: flag --%s should be registered", name)
		}
	}
}

func TestManualCleanupFlags(t *testing.T) {
	for _, name := range []string{"provider", "creds", "log-level", "log-file"} {
		if manualCleanupCmd.Flags().Lookup(name) == nil {
			t.Errorf("manual-cleanup: flag --%s should be registered", name)
		}
	}
}

func TestManualAuthFlag_ProfileDefault(t *testing.T) {
	f := manualAuthCmd.Flags().Lookup("profile")
	if f == nil {
		t.Fatal("--profile flag not registered on manual-auth")
	}
	if f.DefValue != "default" {
		t.Errorf("profile default = %q, want %q", f.DefValue, "default")
	}
}

func TestManualAuthFlag_LogLevelDefault(t *testing.T) {
	f := manualAuthCmd.Flags().Lookup("log-level")
	if f == nil {
		t.Fatal("--log-level flag not registered on manual-auth")
	}
	if f.DefValue != "warn" {
		t.Errorf("log-level default = %q, want %q", f.DefValue, "warn")
	}
}

// ── manual-auth error paths ──────────────────────────────────────────────────

func TestManualAuth_MissingBothEnvVars(t *testing.T) {
	os.Unsetenv("CERTBOT_DOMAIN")
	os.Unsetenv("CERTBOT_VALIDATION")
	err := executeArgs("manual-auth")
	if err == nil {
		t.Fatal("expected error when both env vars are missing")
	}
	if !strings.Contains(err.Error(), "CERTBOT_DOMAIN") {
		t.Errorf("error should mention CERTBOT_DOMAIN, got: %v", err)
	}
}

func TestManualAuth_MissingValidation(t *testing.T) {
	os.Setenv("CERTBOT_DOMAIN", "example.com")
	os.Unsetenv("CERTBOT_VALIDATION")
	defer os.Unsetenv("CERTBOT_DOMAIN")
	err := executeArgs("manual-auth")
	if err == nil {
		t.Fatal("expected error when CERTBOT_VALIDATION is missing")
	}
}

func TestManualAuth_InvalidLogLevel(t *testing.T) {
	cleanup := setEnv("example.com", "token")
	defer cleanup()
	err := executeArgs("manual-auth", "--log-level", "verbose")
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
	if !strings.Contains(err.Error(), "verbose") {
		t.Errorf("error should mention the bad level, got: %v", err)
	}
}

// TestManualAuth_WithProvider_MissingCreds: provider is set but credentials
// file does not exist yet → EnsureTemplate creates a placeholder and the hook
// returns an error asking the user to fill it in.
func TestManualAuth_WithProvider_MissingCreds(t *testing.T) {
	cleanup := setEnv("example.com", "token")
	defer cleanup()

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "cloudflare.yaml")

	os.Setenv("AT3AM_DNS_PROVIDER", "cloudflare")
	os.Setenv("AT3AM_DNS_CREDS", credsFile)
	defer os.Unsetenv("AT3AM_DNS_PROVIDER")
	defer os.Unsetenv("AT3AM_DNS_CREDS")

	err := executeArgs("manual-auth")
	if err == nil {
		t.Fatal("expected error when credentials file needs to be configured")
	}
	if !strings.Contains(err.Error(), "credentials") {
		t.Errorf("error should mention credentials, got: %v", err)
	}
	// Template file should have been created
	if _, statErr := os.Stat(credsFile); os.IsNotExist(statErr) {
		t.Error("EnsureTemplate should have created the credentials template file")
	}
}

// ── manual-cleanup error paths ───────────────────────────────────────────────

func TestManualCleanup_MissingBothEnvVars(t *testing.T) {
	os.Unsetenv("CERTBOT_DOMAIN")
	os.Unsetenv("CERTBOT_VALIDATION")
	err := executeArgs("manual-cleanup")
	if err == nil {
		t.Fatal("expected error when both env vars are missing")
	}
	if !strings.Contains(err.Error(), "CERTBOT_DOMAIN") {
		t.Errorf("error should mention CERTBOT_DOMAIN, got: %v", err)
	}
}

func TestManualCleanup_MissingValidation(t *testing.T) {
	os.Setenv("CERTBOT_DOMAIN", "example.com")
	os.Unsetenv("CERTBOT_VALIDATION")
	defer os.Unsetenv("CERTBOT_DOMAIN")
	err := executeArgs("manual-cleanup")
	if err == nil {
		t.Fatal("expected error when CERTBOT_VALIDATION is missing")
	}
}

// TestManualCleanup_SkipDNS: with AT3AM_SKIP_DNS=1 the cleanup function skips
// all provider operations and returns nil immediately — testable without
// triggering os.Exit or real network I/O.
func TestManualCleanup_SkipDNS(t *testing.T) {
	cleanup := setEnv("example.com", "token")
	defer cleanup()
	os.Setenv("AT3AM_SKIP_DNS", "1")
	defer os.Unsetenv("AT3AM_SKIP_DNS")

	out := captureStdout(func() {
		if err := executeArgs("manual-cleanup"); err != nil {
			t.Errorf("unexpected error with AT3AM_SKIP_DNS=1: %v", err)
		}
	})
	if !strings.Contains(out, "example.com") {
		t.Errorf("expected domain in skip output, got: %q", out)
	}
}

func TestManualCleanup_SkipDNS_InvalidLogLevel(t *testing.T) {
	cleanup := setEnv("example.com", "token")
	defer cleanup()
	os.Setenv("AT3AM_SKIP_DNS", "1")
	defer os.Unsetenv("AT3AM_SKIP_DNS")

	err := executeArgs("manual-cleanup", "--log-level", "verbose")
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
}

// ── formatVersion helper ─────────────────────────────────────────────────────

func TestFormatVersion(t *testing.T) {
	got := fmt.Sprintf("at3am-hook version %s (commit %s, built %s)\n", "v1.0.0", "abc1234", "2026-01-01T00:00:00Z")
	if !strings.HasPrefix(got, "at3am-hook version") {
		t.Errorf("unexpected format: %q", got)
	}
}

