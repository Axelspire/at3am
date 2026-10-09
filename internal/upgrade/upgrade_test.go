package upgrade

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeModule(t *testing.T) {
	if got := normalizeModule(" v0.2.2\n"); got != "0.2.2" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeModule("0.2.2"); got != "0.2.2" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadAgentEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AT3AM_CONFIG_DIR", dir)
	t.Setenv("THREEAM_AGENT_BASE", "")
	t.Setenv("AT3AM_MODULE", "")
	path := filepath.Join(dir, "agent.env")
	content := "THREEAM_AGENT_BASE=https://axel.example/root/agent/at3am\nAT3AM_MODULE=v0.2.2\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	base, mod, err := loadAgentEnv()
	if err != nil {
		t.Fatal(err)
	}
	if base != "https://axel.example/root/agent/at3am" {
		t.Fatalf("base=%q", base)
	}
	if mod != "v0.2.2" {
		t.Fatalf("module=%q", mod)
	}
}

func TestMaybeSelfUpgradeSkippedWithoutBase(t *testing.T) {
	t.Setenv("AT3AM_NO_UPGRADE", "")
	t.Setenv("THREEAM_AGENT_BASE", "")
	t.Setenv("AT3AM_MODULE", "")
	t.Setenv("AT3AM_CONFIG_DIR", t.TempDir())
	reexec, err := MaybeSelfUpgrade("v0.2.2")
	if err != nil {
		t.Fatal(err)
	}
	if reexec {
		t.Fatal("expected no reexec without agent base")
	}
}

func TestMaybeSelfUpgradeHonoursNoUpgrade(t *testing.T) {
	t.Setenv("AT3AM_NO_UPGRADE", "1")
	t.Setenv("THREEAM_AGENT_BASE", "https://example.invalid")
	reexec, err := MaybeSelfUpgrade("v0.2.2")
	if err != nil {
		t.Fatal(err)
	}
	if reexec {
		t.Fatal("expected skip when AT3AM_NO_UPGRADE=1")
	}
}
