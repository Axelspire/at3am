package log

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetLoggers saves and restores global logger state around each test so tests
// do not bleed into one another.
func resetLoggers(t *testing.T) {
	t.Helper()
	origConsoleOut := console.out
	origConsoleLevel := console.level
	origFileOut := filelog.out
	origFileLevel := filelog.level
	t.Cleanup(func() {
		console.mu.Lock()
		console.out = origConsoleOut
		console.level = origConsoleLevel
		console.mu.Unlock()
		filelog.mu.Lock()
		filelog.out = origFileOut
		filelog.level = origFileLevel
		filelog.mu.Unlock()
	})
}

// ── ParseLevel ────────────────────────────────────────────────────────────────

func TestParseLevel_ValidInputs(t *testing.T) {
	cases := []struct {
		in   string
		want Level
	}{
		{"debug", DEBUG},
		{"DEBUG", DEBUG},
		{"info", INFO},
		{"INFO", INFO},
		{"warn", WARN},
		{"warning", WARN},
		{"WARN", WARN},
		{"error", ERROR},
		{"ERROR", ERROR},
		{"off", OFF},
		{"OFF", OFF},
		{"", OFF},
		{"  info  ", INFO}, // trimmed
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseLevel(tc.in)
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ParseLevel(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseLevel_Invalid(t *testing.T) {
	_, err := ParseLevel("verbose")
	if err == nil {
		t.Error("expected error for unknown level, got nil")
	}
	if !strings.Contains(err.Error(), "verbose") {
		t.Errorf("error should mention the bad value, got: %v", err)
	}
}

// ── Init ─────────────────────────────────────────────────────────────────────

func TestInit_ConsoleOnly(t *testing.T) {
	resetLoggers(t)

	teardown, err := Init(INFO, "")
	if err != nil {
		t.Fatalf("Init error: %v", err)
	}
	defer teardown()

	if console.out == nil {
		t.Error("console.out should be set after Init with INFO level")
	}
	if console.level != INFO {
		t.Errorf("console.level = %v, want INFO", console.level)
	}
	if filelog.out != nil {
		t.Error("filelog.out should be nil when no file path given")
	}
}

func TestInit_WithLogFile(t *testing.T) {
	resetLoggers(t)

	path := filepath.Join(t.TempDir(), "test.log")
	teardown, err := Init(WARN, path)
	if err != nil {
		t.Fatalf("Init error: %v", err)
	}
	defer teardown()

	if filelog.out == nil {
		t.Error("filelog.out should be set when a file path is given")
	}
}

func TestInit_InvalidFilePath(t *testing.T) {
	resetLoggers(t)

	_, err := Init(INFO, "/nonexistent_dir/does_not_exist/test.log")
	if err == nil {
		t.Error("expected error for invalid log file path, got nil")
	}
}

func TestInit_Teardown_ClosesFile(t *testing.T) {
	resetLoggers(t)

	path := filepath.Join(t.TempDir(), "teardown.log")
	teardown, err := Init(INFO, path)
	if err != nil {
		t.Fatalf("Init error: %v", err)
	}
	teardown()

	if filelog.out != nil {
		t.Error("filelog.out should be nil after teardown")
	}
}

// ── Emit routing ──────────────────────────────────────────────────────────────

func TestDebug_ConsoleOnly(t *testing.T) {
	resetLoggers(t)
	consoleBuf := &bytes.Buffer{}
	fileBuf := &bytes.Buffer{}
	console.out = consoleBuf
	console.level = DEBUG
	filelog.out = fileBuf
	filelog.level = INFO

	Debug("hello %s", "debug")

	if !strings.Contains(consoleBuf.String(), "DEBUG") {
		t.Errorf("DEBUG not in console output: %q", consoleBuf.String())
	}
	if fileBuf.Len() != 0 {
		t.Errorf("DEBUG should not reach file log, got: %q", fileBuf.String())
	}
}

func TestInfo_ConsoleAndFile(t *testing.T) {
	resetLoggers(t)
	consoleBuf := &bytes.Buffer{}
	fileBuf := &bytes.Buffer{}
	console.out = consoleBuf
	console.level = INFO
	filelog.out = fileBuf
	filelog.level = INFO

	Info("hello %s", "info")

	if !strings.Contains(consoleBuf.String(), "INFO") {
		t.Errorf("INFO not in console output: %q", consoleBuf.String())
	}
	if !strings.Contains(fileBuf.String(), "INFO") {
		t.Errorf("INFO not in file output: %q", fileBuf.String())
	}
}

func TestWarn_ConsoleAndFile(t *testing.T) {
	resetLoggers(t)
	consoleBuf := &bytes.Buffer{}
	fileBuf := &bytes.Buffer{}
	console.out = consoleBuf
	console.level = WARN
	filelog.out = fileBuf
	filelog.level = INFO

	Warn("something bad")

	if !strings.Contains(consoleBuf.String(), "WARN") {
		t.Errorf("WARN not in console output: %q", consoleBuf.String())
	}
	if !strings.Contains(fileBuf.String(), "WARN") {
		t.Errorf("WARN not in file output: %q", fileBuf.String())
	}
}

func TestError_ConsoleAndFile(t *testing.T) {
	resetLoggers(t)
	consoleBuf := &bytes.Buffer{}
	fileBuf := &bytes.Buffer{}
	console.out = consoleBuf
	console.level = ERROR
	filelog.out = fileBuf
	filelog.level = INFO

	Error("fatal %d", 42)

	if !strings.Contains(consoleBuf.String(), "ERROR") {
		t.Errorf("ERROR not in console output: %q", consoleBuf.String())
	}
	if !strings.Contains(fileBuf.String(), "ERROR") {
		t.Errorf("ERROR not in file output: %q", fileBuf.String())
	}
}

func TestLevel_Filtering(t *testing.T) {
	resetLoggers(t)
	buf := &bytes.Buffer{}
	console.out = buf
	console.level = ERROR // only ERROR and above

	Debug("should not appear")
	Info("should not appear")
	Warn("should not appear")
	Error("should appear")

	out := buf.String()
	if strings.Contains(out, "DEBUG") || strings.Contains(out, "INFO") || strings.Contains(out, "WARN") {
		t.Errorf("messages below ERROR should be filtered, got: %q", out)
	}
	if !strings.Contains(out, "ERROR") {
		t.Errorf("ERROR should be present, got: %q", out)
	}
}

func TestEmit_SilentWhenOff(t *testing.T) {
	resetLoggers(t)
	buf := &bytes.Buffer{}
	console.out = buf
	console.level = OFF

	Info("this should not appear")
	Warn("this should not appear")
	Error("this should not appear")

	if buf.Len() != 0 {
		t.Errorf("expected no output at OFF level, got: %q", buf.String())
	}
}

// ── Log format ────────────────────────────────────────────────────────────────

func TestLogFormat(t *testing.T) {
	resetLoggers(t)
	buf := &bytes.Buffer{}
	console.out = buf
	console.level = INFO

	Info("test-message")

	line := buf.String()
	// Expected format: 2006-01-02 15:04:05.000:INFO:pkg/file.go:123:test-message\n
	// The timestamp itself contains colons so we check each component via Contains.
	if !strings.Contains(line, ":INFO:") {
		t.Errorf("level label :INFO: not found in output: %q", line)
	}
	if !strings.Contains(line, "test-message") {
		t.Errorf("message not found in output: %q", line)
	}
	if !strings.Contains(line, "log_test.go:") {
		t.Errorf("caller file not found in output: %q", line)
	}
	if !strings.HasSuffix(line, "\n") {
		t.Error("log line should end with newline")
	}
}

func TestLogFormat_NoArgs(t *testing.T) {
	resetLoggers(t)
	buf := &bytes.Buffer{}
	console.out = buf
	console.level = INFO

	Info("plain message no format args")

	if !strings.Contains(buf.String(), "plain message no format args") {
		t.Errorf("plain message not found in output: %q", buf.String())
	}
}

// ── Init with OFF console level ───────────────────────────────────────────────

func TestInit_OffLevel_NoConsoleOutput(t *testing.T) {
	resetLoggers(t)

	teardown, err := Init(OFF, "")
	if err != nil {
		t.Fatalf("Init error: %v", err)
	}
	defer teardown()

	// console.out is set to os.Stdout by Init only when level < OFF
	// so at OFF level it should still be nil (original value)
	buf := &bytes.Buffer{}
	console.out = buf
	Info("should not appear")

	if buf.Len() != 0 {
		t.Errorf("expected no output at OFF level, got: %q", buf.String())
	}
}

// ── Integration: Init writes to real file ─────────────────────────────────────

func TestInit_FileReceivesInfoNotDebug(t *testing.T) {
	resetLoggers(t)

	path := filepath.Join(t.TempDir(), "routing.log")
	teardown, err := Init(DEBUG, path)
	if err != nil {
		t.Fatalf("Init error: %v", err)
	}
	defer teardown()

	// Write via the real file handle by replacing console with discard so only
	// file output matters for this assertion.
	console.out = &bytes.Buffer{}

	Debug("debug-only")
	Info("info-to-file")

	teardown() // flush/close before reading

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "debug-only") {
		t.Error("DEBUG should not appear in file log")
	}
	if !strings.Contains(content, "info-to-file") {
		t.Errorf("INFO should appear in file log, got: %q", content)
	}
}

