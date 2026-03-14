package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Timeout != 300*time.Second {
		t.Errorf("expected timeout 300s, got %s", cfg.Timeout)
	}
	if cfg.Interval != 5*time.Second {
		t.Errorf("expected interval 5s, got %s", cfg.Interval)
	}
	if cfg.Threshold != 95.0 {
		t.Errorf("expected threshold 95.0, got %.1f", cfg.Threshold)
	}
	if cfg.ConsecutivePasses != 2 {
		t.Errorf("expected 2 consecutive passes, got %d", cfg.ConsecutivePasses)
	}
	if cfg.Profile != ProfileDefault {
		t.Errorf("expected profile 'default', got %q", cfg.Profile)
	}
	if cfg.OutputFormat != "human" {
		t.Errorf("expected output format 'human', got %q", cfg.OutputFormat)
	}
	if cfg.AuthWeight != 0.6 {
		t.Errorf("expected auth weight 0.6, got %.2f", cfg.AuthWeight)
	}
	if cfg.PublicWeight != 0.4 {
		t.Errorf("expected public weight 0.4, got %.2f", cfg.PublicWeight)
	}
}

func TestApplyProfile(t *testing.T) {
	tests := []struct {
		profile           Profile
		expectedThreshold float64
		expectedPasses    int
		expectedInterval  time.Duration
		expectedTimeout   time.Duration
		expectErr         bool
	}{
		{ProfileStrict, 100.0, 3, 10 * time.Second, 600 * time.Second, false},
		{ProfileDefault, 95.0, 2, 5 * time.Second, 300 * time.Second, false},
		{ProfileFast, 80.0, 1, 2 * time.Second, 120 * time.Second, false},
		{ProfileYolo, 50.0, 1, 1 * time.Second, 60 * time.Second, false},
		{Profile("unknown"), 0, 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.profile), func(t *testing.T) {
			cfg := DefaultConfig()
			err := cfg.ApplyProfile(tt.profile)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Threshold != tt.expectedThreshold {
				t.Errorf("threshold: got %.1f, want %.1f", cfg.Threshold, tt.expectedThreshold)
			}
			if cfg.ConsecutivePasses != tt.expectedPasses {
				t.Errorf("passes: got %d, want %d", cfg.ConsecutivePasses, tt.expectedPasses)
			}
			if cfg.Interval != tt.expectedInterval {
				t.Errorf("interval: got %s, want %s", cfg.Interval, tt.expectedInterval)
			}
			if cfg.Timeout != tt.expectedTimeout {
				t.Errorf("timeout: got %s, want %s", cfg.Timeout, tt.expectedTimeout)
			}
			if cfg.Profile != tt.profile {
				t.Errorf("profile: got %q, want %q", cfg.Profile, tt.profile)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		modifier  func(*Config)
		expectErr bool
		errMsg    string
	}{
		{"valid", func(c *Config) { c.Domain = "example.com"; c.Expected = "token" }, false, ""},
		{"missing domain", func(c *Config) { c.Expected = "token" }, true, "domain is required"},
		{"missing expected", func(c *Config) { c.Domain = "example.com" }, true, "expected TXT value is required"},
		{"threshold too low", func(c *Config) { c.Domain = "x"; c.Expected = "y"; c.Threshold = -1 }, true, "threshold must be"},
		{"threshold too high", func(c *Config) { c.Domain = "x"; c.Expected = "y"; c.Threshold = 101 }, true, "threshold must be"},
		{"passes too low", func(c *Config) { c.Domain = "x"; c.Expected = "y"; c.ConsecutivePasses = 0 }, true, "consecutive passes"},
		{"interval too low", func(c *Config) { c.Domain = "x"; c.Expected = "y"; c.Interval = 500 * time.Millisecond }, true, "interval must be"},
		{"timeout < interval", func(c *Config) { c.Domain = "x"; c.Expected = "y"; c.Timeout = 1 * time.Second; c.Interval = 2 * time.Second }, true, "timeout"},
		{"auth weight negative", func(c *Config) { c.Domain = "x"; c.Expected = "y"; c.AuthWeight = -0.1 }, true, "auth-weight"},
		{"auth weight > 1", func(c *Config) { c.Domain = "x"; c.Expected = "y"; c.AuthWeight = 1.1 }, true, "auth-weight"},
		{"public weight negative", func(c *Config) { c.Domain = "x"; c.Expected = "y"; c.PublicWeight = -0.1 }, true, "public-weight"},
		{"public weight > 1", func(c *Config) { c.Domain = "x"; c.Expected = "y"; c.PublicWeight = 1.1 }, true, "public-weight"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modifier(&cfg)
			err := cfg.Validate()
			if tt.expectErr {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

