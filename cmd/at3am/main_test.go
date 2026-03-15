package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
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
func resetFlags() {
	waitCmd.Flags().Set("profile", "default")
	waitCmd.Flags().Set("output", "text")
	waitCmd.Flags().Set("log-level", "warn")
	waitCmd.Flags().Set("log-file", "")
	waitCmd.Flags().Set("challenge-type", "dns-01")
	waitCmd.Flags().Set("mock-scenario", "instant")
	waitCmd.Flags().Set("dnssec-validate", "false")
	waitCmd.Flags().Set("mock", "false")
	waitCmd.Flags().Set("timeout", "5m0s")
	waitCmd.Flags().Set("interval", "5s")
	waitCmd.Flags().Set("resolvers", "")
	waitCmd.Flags().Set("on-ready", "")
	waitCmd.Flags().Set("webhook", "")
	waitCmd.Flags().Set("prometheus-port", "0")
}

// executeArgs resets all flags, sets rootCmd args, and executes.
func executeArgs(args ...string) error {
	resetFlags()
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

// ── version command ──────────────────────────────────────────────────────────

func TestVersionCommand_Output(t *testing.T) {
	out := captureStdout(func() {
		executeArgs("version")
	})
	if !strings.Contains(out, "at3am version") {
		t.Errorf("expected 'at3am version' in output, got: %q", out)
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
	for _, want := range []string{"version", "wait"} {
		if !names[want] {
			t.Errorf("expected command %q to be registered", want)
		}
	}
}

func TestWaitCommand_RequiredFlags(t *testing.T) {
	if !waitCmd.HasFlags() {
		t.Fatal("wait command should have flags")
	}
	for _, name := range []string{"domain", "expected"} {
		f := waitCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("flag --%s should be registered", name)
		}
	}
}

// ── flag defaults ────────────────────────────────────────────────────────────

func TestWaitFlagDefaults(t *testing.T) {
	cases := []struct {
		flag string
		want string
	}{
		{"profile", "default"},
		{"output", "text"},
		{"log-level", "warn"},
		{"challenge-type", "dns-01"},
		{"mock-scenario", "instant"},
	}
	for _, tc := range cases {
		f := waitCmd.Flags().Lookup(tc.flag)
		if f == nil {
			t.Errorf("flag --%s not registered", tc.flag)
			continue
		}
		if f.DefValue != tc.want {
			t.Errorf("--%s default = %q, want %q", tc.flag, f.DefValue, tc.want)
		}
	}
}

func TestWaitFlagDefaults_Duration(t *testing.T) {
	timeout, _ := waitCmd.Flags().GetDuration("timeout")
	if timeout != 5*time.Minute {
		t.Errorf("timeout default = %s, want 5m0s", timeout)
	}
	interval, _ := waitCmd.Flags().GetDuration("interval")
	if interval != 5*time.Second {
		t.Errorf("interval default = %s, want 5s", interval)
	}
}

func TestWaitFlagDefaults_Bool(t *testing.T) {
	dnssec, _ := waitCmd.Flags().GetBool("dnssec-validate")
	if dnssec {
		t.Error("dnssec-validate default should be false")
	}
	mock, _ := waitCmd.Flags().GetBool("mock")
	if mock {
		t.Error("mock default should be false")
	}
}

// ── error paths ──────────────────────────────────────────────────────────────

func TestWaitCommand_InvalidLogLevel(t *testing.T) {
	err := executeArgs("wait", "--domain", "_acme-challenge.example.com", "--expected", "token", "--log-level", "verbose")
	if err == nil {
		t.Error("expected error for invalid log level")
	}
	if !strings.Contains(err.Error(), "verbose") {
		t.Errorf("error should mention the bad log level, got: %v", err)
	}
}

func TestWaitCommand_InvalidProfile(t *testing.T) {
	err := executeArgs("wait", "--domain", "_acme-challenge.example.com", "--expected", "token", "--profile", "badprofile")
	if err == nil {
		t.Error("expected error for invalid profile")
	}
}

func TestWaitCommand_InvalidLogFile(t *testing.T) {
	err := executeArgs("wait", "--domain", "_acme-challenge.example.com", "--expected", "token", "--log-file", "/nonexistent/path/at3am.log")
	if err == nil {
		t.Error("expected error for unwritable log file")
	}
}

// ── formatVersion helper ─────────────────────────────────────────────────────

func TestFormatVersion(t *testing.T) {
	got := fmt.Sprintf("at3am version %s (commit %s, built %s)\n", "v1.0.0", "abc1234", "2026-01-01T00:00:00Z")
	if !strings.HasPrefix(got, "at3am version") {
		t.Errorf("unexpected format: %q", got)
	}
}

