// Package ttl implements TTL analysis for DNS propagation monitoring.
package ttl

import (
	"fmt"
	"time"

	"github.com/axelspire/at3am/internal/resolver"
)

// Report represents a TTL audit report.
type Report struct {
	// MinTTL is the minimum TTL observed across all resolvers.
	MinTTL uint32 `json:"min_ttl"`
	// MaxTTL is the maximum TTL observed across all resolvers.
	MaxTTL uint32 `json:"max_ttl"`
	// AvgTTL is the average TTL observed.
	AvgTTL uint32 `json:"avg_ttl"`
	// TTLValues maps resolver addresses to their observed TTLs.
	TTLValues map[string]uint32 `json:"ttl_values"`
	// EstimatedFullPropagation is the estimated time for full propagation.
	EstimatedFullPropagation time.Duration `json:"estimated_full_propagation"`
	// Warnings contains any TTL-related warnings.
	Warnings []string `json:"warnings,omitempty"`
	// SampleCount is the number of resolvers that returned TTL values.
	SampleCount int `json:"sample_count"`
}

// Analyser performs TTL analysis on DNS query results.
type Analyser struct{}

// NewAnalyser creates a new TTL analyser.
func NewAnalyser() *Analyser {
	return &Analyser{}
}

// Analyse examines resolver results and produces a TTL report.
func (a *Analyser) Analyse(results []resolver.Result) Report {
	report := Report{
		TTLValues: make(map[string]uint32),
	}

	var totalTTL uint64
	first := true

	for _, r := range results {
		if r.Error != "" || !r.Found {
			continue
		}

		report.TTLValues[r.Resolver] = r.TTL
		report.SampleCount++
		totalTTL += uint64(r.TTL)

		if first {
			report.MinTTL = r.TTL
			report.MaxTTL = r.TTL
			first = false
		} else {
			if r.TTL < report.MinTTL {
				report.MinTTL = r.TTL
			}
			if r.TTL > report.MaxTTL {
				report.MaxTTL = r.TTL
			}
		}
	}

	if report.SampleCount > 0 {
		report.AvgTTL = uint32(totalTTL / uint64(report.SampleCount))
		report.EstimatedFullPropagation = time.Duration(report.MaxTTL) * time.Second
	}

	// Generate warnings
	report.Warnings = a.generateWarnings(report)
	return report
}

func (a *Analyser) generateWarnings(report Report) []string {
	var warnings []string

	if report.SampleCount == 0 {
		warnings = append(warnings, "No TTL data available — record not found by any resolver")
		return warnings
	}

	if report.MaxTTL > 3600 {
		warnings = append(warnings, fmt.Sprintf(
			"High TTL detected (%ds = %s). Propagation may be slow due to aggressive caching.",
			report.MaxTTL, time.Duration(report.MaxTTL)*time.Second,
		))
	}

	if report.MinTTL < 30 {
		warnings = append(warnings, fmt.Sprintf(
			"Very low TTL (%ds). This is fine for ACME challenges but may indicate misconfiguration for production records.",
			report.MinTTL,
		))
	}

	if report.MaxTTL > 0 && report.MinTTL > 0 {
		spread := report.MaxTTL - report.MinTTL
		if spread > report.MaxTTL/2 {
			warnings = append(warnings, fmt.Sprintf(
				"Large TTL spread detected (min=%ds, max=%ds). Different resolvers have very different cache ages.",
				report.MinTTL, report.MaxTTL,
			))
		}
	}

	return warnings
}

