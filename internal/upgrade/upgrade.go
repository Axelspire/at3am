// Package upgrade implements vanity self-upgrade for at3am (CORE-62).
// When THREEAM_AGENT_BASE (or ~/.config/at3am/agent.env) is set, every run
// compares the baked version to $AGENT_BASE/MODULE and refreshes the install
// tree before continuing. Set AT3AM_NO_UPGRADE=1 to skip.
package upgrade

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

// MaybeSelfUpgrade checks the vanity MODULE and re-runs install.sh/ps1 when
// drifted. Returns true if this process should exit (child will continue).
func MaybeSelfUpgrade(currentVersion string) (reexec bool, err error) {
	if os.Getenv("AT3AM_NO_UPGRADE") == "1" {
		return false, nil
	}
	base, have, err := loadAgentEnv()
	if err != nil || base == "" {
		return false, nil // community / no vanity — skip
	}
	want, err := fetchText(strings.TrimRight(base, "/") + "/MODULE")
	if err != nil {
		// Network blip: do not block the run.
		fmt.Fprintf(os.Stderr, "at3am: upgrade check skipped: %v\n", err)
		return false, nil
	}
	want = normalizeModule(want)
	have = normalizeModule(have)
	cur := normalizeModule(currentVersion)
	if have == "" {
		have = cur
	}
	if want == "" || want == have || want == cur {
		return false, nil
	}

	fmt.Fprintf(os.Stderr, "at3am: module %s → %s; self-upgrading…\n", have, want)
	if err := runInstall(base); err != nil {
		return false, fmt.Errorf("self-upgrade: %w", err)
	}
	return true, reexecSelf()
}

func normalizeModule(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	return s
}

func loadAgentEnv() (base, module string, err error) {
	if v := strings.TrimSpace(os.Getenv("THREEAM_AGENT_BASE")); v != "" {
		base = v
	}
	if v := strings.TrimSpace(os.Getenv("AT3AM_MODULE")); v != "" {
		module = v
	}
	cfg := os.Getenv("AT3AM_CONFIG_DIR")
	if cfg == "" {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return base, module, herr
		}
		cfg = filepath.Join(home, ".config", "at3am")
	}
	path := filepath.Join(cfg, "agent.env")
	data, rerr := os.ReadFile(path)
	if rerr != nil {
		if os.IsNotExist(rerr) {
			return base, module, nil
		}
		return base, module, rerr
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "THREEAM_AGENT_BASE":
			if base == "" {
				base = val
			}
		case "AT3AM_MODULE":
			if module == "" {
				module = val
			}
		}
	}
	return base, module, nil
}

func fetchText(url string) (string, error) {
	client := &http.Client{Timeout: defaultTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func runInstall(base string) error {
	base = strings.TrimRight(base, "/")
	client := &http.Client{Timeout: 2 * time.Minute}
	if runtime.GOOS == "windows" {
		url := base + "/install.ps1"
		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
		}
		script, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		tmp, err := os.CreateTemp("", "at3am-install-*.ps1")
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name())
		if _, err := tmp.Write(script); err != nil {
			tmp.Close()
			return err
		}
		tmp.Close()
		cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", tmp.Name())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	url := base + "/install.sh"
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	cmd := exec.Command("sh")
	cmd.Stdin = resp.Body
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func reexecSelf() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// Prefer lib binary after install refresh.
	lib := os.Getenv("AT3AM_LIB_DIR")
	if lib == "" {
		home, herr := os.UserHomeDir()
		if herr == nil {
			if runtime.GOOS == "windows" {
				lib = filepath.Join(os.Getenv("LOCALAPPDATA"), "3am", "at3am")
			} else {
				lib = filepath.Join(home, ".local", "lib", "3am", "at3am")
			}
		}
	}
	cand := filepath.Join(lib, "at3am")
	if runtime.GOOS == "windows" {
		cand += ".exe"
	}
	if st, err := os.Stat(cand); err == nil && !st.IsDir() {
		exe = cand
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = append(os.Environ(), "AT3AM_NO_UPGRADE=1")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			os.Exit(ee.ExitCode())
		}
		return err
	}
	os.Exit(0)
	return nil
}
