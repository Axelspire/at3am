// Package config defines configuration types and profiles for at3am.
package config

import (
	"fmt"
	"time"
)

// Profile represents a pre-built confidence profile.
type Profile string

const (
	ProfileStrict  Profile = "strict"
	ProfileDefault Profile = "default"
	ProfileFast    Profile = "fast"
	ProfileYolo    Profile = "yolo"
)

// ChallengeType selects the ACME challenge variant to monitor.
type ChallengeType string

const (
	// ChallengeTypeDNS01 is the standard DNS-01 challenge: a record is
	// provisioned per validation and removed afterwards. Auth-first gating
	// is applied to avoid negative-caching on public resolvers.
	ChallengeTypeDNS01 ChallengeType = "dns-01"

	// ChallengeTypePersist is the DNS-PERSIST-01 challenge (IETF draft):
	// a TXT record is pre-provisioned permanently. All resolvers are queried
	// simultaneously without auth-first gating, because the record is
	// expected to already be universally reachable.
	ChallengeTypePersist ChallengeType = "persist"
)

// Config holds all configuration for an at3am run.
type Config struct {
	// Domain is the FQDN to check (e.g., _acme-challenge.example.com).
	Domain string
	// Expected is the expected TXT record value.
	Expected string
	// Timeout is the maximum time to wait for propagation.
	Timeout time.Duration
	// Interval is the polling interval between checks.
	Interval time.Duration
	// Threshold is the confidence threshold (0-100).
	Threshold float64
	// ConsecutivePasses is the number of consecutive passes required.
	ConsecutivePasses int
	// Profile is the pre-built profile name.
	Profile Profile
	// Resolvers is a list of additional custom resolvers.
	Resolvers []string
	// OutputFormat is the output format (human, json, quiet).
	OutputFormat string
	// PrometheusPort is the port for Prometheus metrics (0 = disabled).
	PrometheusPort int
	// OnReady is a command to execute when propagation is confirmed.
	OnReady string
	// WebhookURL is a URL to POST to when propagation is confirmed.
	WebhookURL string
	// MockMode enables mock DNS resolution for testing.
	MockMode bool
	// MockScenario selects a specific mock scenario.
	MockScenario string
	// LogLevel sets the minimum severity for stdout log output.
	// Values: "debug", "info", "warn", "error", "" (default — no stdout logs).
	// --debug is a shorthand that sets this to "debug".
	LogLevel string
	// LogFile is the path to write production log output (INFO+ only).
	// When set, each poll summary, ready events, and final latency are appended.
	// Empty means no file logging.
	LogFile string
	// AuthWeight is the weight for authoritative NS results (0-1).
	AuthWeight float64
	// PublicWeight is the weight for public resolver results (0-1).
	PublicWeight float64
	// AuthGateTimeout is the max time to wait for auth NS confirmation before
	// falling back to public resolvers (0 = no timeout, always gate).
	AuthGateTimeout time.Duration
	// DNSSECValidate enables DNSSEC validation checks. When true, the DO bit
	// is set on queries and the AD (Authenticated Data) bit in responses is
	// recorded. Results are surfaced in output but do not block readiness.
	DNSSECValidate bool
	// ChallengeType selects the ACME challenge variant (dns-01 or persist).
	// Defaults to dns-01.
	ChallengeType ChallengeType
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Timeout:           300 * time.Second,
		Interval:          3 * time.Second,
		Threshold:         95.0,
		ConsecutivePasses: 2,
		Profile:           ProfileDefault,
		OutputFormat:      "human",
		AuthWeight:        0.6,
		PublicWeight:      0.4,
		AuthGateTimeout:   180 * time.Second, // 3 minutes
		ChallengeType:     ChallengeTypeDNS01,
	}
}

// ApplyProfile applies a named profile's settings to the config.
func (c *Config) ApplyProfile(p Profile) error {
	switch p {
	case ProfileStrict:
		// Auth threshold: ALL, Public threshold: ALL minus 2, Consecutive: 3, Timeout: 600s
		c.Threshold = 100.0
		c.ConsecutivePasses = 3
		c.Interval = 3 * time.Second
		c.Timeout = 600 * time.Second
	case ProfileDefault:
		// Auth threshold: ALL, Public threshold: 1, Consecutive: 2, Timeout: 300s
		c.Threshold = 95.0
		c.ConsecutivePasses = 2
		c.Interval = 3 * time.Second
		c.Timeout = 300 * time.Second
	case ProfileFast:
		// Auth threshold: ≥ ceil(N/2), Public threshold: 1, Consecutive: 1, Timeout: 120s
		c.Threshold = 80.0
		c.ConsecutivePasses = 1
		c.Interval = 2 * time.Second
		c.Timeout = 120 * time.Second
	case ProfileYolo:
		// Auth threshold: 1, Public threshold: 0, Consecutive: 1, Timeout: 60s
		c.Threshold = 50.0
		c.ConsecutivePasses = 1
		c.Interval = 1 * time.Second
		c.Timeout = 60 * time.Second
	default:
		return fmt.Errorf("unknown profile: %s", p)
	}
	c.Profile = p
	return nil
}

// Validate checks that the config is valid.
func (c *Config) Validate() error {
	if c.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if c.Expected == "" {
		return fmt.Errorf("expected TXT value is required")
	}
	if c.Threshold < 0 || c.Threshold > 100 {
		return fmt.Errorf("threshold must be between 0 and 100, got %.1f", c.Threshold)
	}
	if c.ConsecutivePasses < 1 {
		return fmt.Errorf("consecutive passes must be at least 1, got %d", c.ConsecutivePasses)
	}
	if c.Interval < 1*time.Second {
		return fmt.Errorf("interval must be at least 1s, got %s", c.Interval)
	}
	if c.Timeout < c.Interval {
		return fmt.Errorf("timeout (%s) must be >= interval (%s)", c.Timeout, c.Interval)
	}
	if c.AuthWeight < 0 || c.AuthWeight > 1 {
		return fmt.Errorf("auth-weight must be between 0 and 1, got %.2f", c.AuthWeight)
	}
	if c.PublicWeight < 0 || c.PublicWeight > 1 {
		return fmt.Errorf("public-weight must be between 0 and 1, got %.2f", c.PublicWeight)
	}
	if c.ChallengeType != "" && c.ChallengeType != ChallengeTypeDNS01 && c.ChallengeType != ChallengeTypePersist {
		return fmt.Errorf("challenge-type must be %q or %q, got %q", ChallengeTypeDNS01, ChallengeTypePersist, c.ChallengeType)
	}
	return nil
}

