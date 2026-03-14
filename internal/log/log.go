// Package log provides a lightweight, structured logger for at3am.
//
// Two independent output targets are managed globally:
//
//   - console — writes to stdout at the level set by --log-level / --debug.
//     All four levels (DEBUG, INFO, WARN, ERROR) are routed here.
//
//   - file — writes to a log file at INFO+ only (production events).
//     DEBUG messages are never written to the file; this keeps the file
//     focused on operational status: startup, per-poll summaries, ready
//     events, and final result with total latency.
//
// Format (one line per event):
//
//	2006-01-02 15:04:05.000:LEVEL:pkg/file.go:line:message
package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level represents log severity.
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	OFF // disables all output
)

var levelLabel = [...]string{"DEBUG", "INFO", "WARN", "ERROR"}

// ParseLevel converts a string to a Level.
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return DEBUG, nil
	case "info":
		return INFO, nil
	case "warn", "warning":
		return WARN, nil
	case "error":
		return ERROR, nil
	case "off", "":
		return OFF, nil
	default:
		return OFF, fmt.Errorf("unknown log level %q — use debug, info, warn, or error", s)
	}
}

// logger is a single-destination, level-filtered writer.
type logger struct {
	mu    sync.Mutex
	level Level
	out   io.Writer // nil → disabled
}

func (l *logger) emit(depth int, level Level, format string, args []any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.out == nil || level < l.level {
		return
	}
	ts := time.Now().Format("2006-01-02 15:04:05.000")
	loc := callerLocation(depth + 1)
	var msg string
	if len(args) == 0 {
		msg = format
	} else {
		msg = fmt.Sprintf(format, args...)
	}
	_, _ = fmt.Fprintf(l.out, "%s:%s:%s:%s\n", ts, levelLabel[level], loc, msg)
}

func (l *logger) close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if c, ok := l.out.(io.Closer); ok {
		_ = c.Close()
	}
	l.out = nil
}

// callerLocation returns "pkg/file.go:line" for the frame at depth.
func callerLocation(depth int) string {
	_, file, line, ok := runtime.Caller(depth)
	if !ok {
		return "unknown:0"
	}
	// Trim to "pkg/file.go" — the last two path segments.
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		if prev := strings.LastIndex(file[:idx], "/"); prev >= 0 {
			file = file[prev+1:]
		}
	}
	return fmt.Sprintf("%s:%d", file, line)
}

// ── Global state ──────────────────────────────────────────────────────────────

var (
	console = &logger{level: OFF} // stdout, user-controlled level
	filelog = &logger{level: INFO} // file, always INFO+
)

// Init configures both global loggers. Call once at startup.
//
//   - consoleLevel: minimum level for stdout. Pass OFF to silence.
//   - filePath: path to the production log file. Empty disables file logging.
//
// Returns a teardown func that flushes and closes the log file.
func Init(consoleLevel Level, filePath string) (func(), error) {
	if consoleLevel < OFF {
		console.level = consoleLevel
		console.out = os.Stdout
	}

	if filePath != "" {
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return func() {}, fmt.Errorf("cannot open log file %q: %w", filePath, err)
		}
		filelog.out = f
	}

	return func() { filelog.close() }, nil
}

// ── Package-level logging functions ──────────────────────────────────────────
//
// DEBUG → console only (never written to the production file)
// INFO / WARN / ERROR → console + file

const callDepth = 2 // public func → emit() → runtime.Caller

// Debug logs a debug-level message to stdout only.
func Debug(format string, args ...any) {
	console.emit(callDepth, DEBUG, format, args)
}

// Info logs an informational message to stdout and the production file.
func Info(format string, args ...any) {
	console.emit(callDepth, INFO, format, args)
	filelog.emit(callDepth, INFO, format, args)
}

// Warn logs a warning to stdout and the production file.
func Warn(format string, args ...any) {
	console.emit(callDepth, WARN, format, args)
	filelog.emit(callDepth, WARN, format, args)
}

// Error logs an error to stdout and the production file.
func Error(format string, args ...any) {
	console.emit(callDepth, ERROR, format, args)
	filelog.emit(callDepth, ERROR, format, args)
}

