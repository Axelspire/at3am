package integration

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// loadEnvFile reads a KEY=VALUE env file and returns the key/value pairs.
// Lines starting with # and blank lines are ignored.
// Values may optionally be quoted with single or double quotes.
func loadEnvFile(relPath string) (map[string]string, error) {
	// Resolve the path relative to the repository root.
	// Go tests run with cwd = package directory, so walk up to find root.
	root, err := repoRoot()
	if err != nil {
		return nil, fmt.Errorf("testenv: cannot locate repo root: %w", err)
	}

	path := filepath.Join(root, relPath)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("testenv: cannot open %s: %w", path, err)
	}
	defer f.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 1 {
			continue // no '=' or empty key
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])

		// Strip optional surrounding quotes.
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		env[key] = val
	}
	return env, scanner.Err()
}

// repoRoot finds the module root by walking up from the caller's source file
// until it finds a go.mod file.
func repoRoot() (string, error) {
	_, callerFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	dir := filepath.Dir(callerFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

