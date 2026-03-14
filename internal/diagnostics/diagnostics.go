// Package diagnostics provides scenario-based diagnostic explanations.
package diagnostics

import (
	"fmt"
	"strings"

	"github.com/axelspire/at3am/internal/confidence"
	"github.com/axelspire/at3am/internal/resolver"
	"github.com/axelspire/at3am/internal/ttl"
)

// Diagnosis represents a diagnostic assessment.
type Diagnosis struct {
	// Scenario is a short identifier for the detected scenario.
	Scenario string `json:"scenario"`
	// Summary is a one-line summary.
	Summary string `json:"summary"`
	// Explanation is a detailed explanation of the situation.
	Explanation string `json:"explanation"`
	// Recommendations are actionable suggestions.
	Recommendations []string `json:"recommendations,omitempty"`
	// Severity is the severity level (info, warning, error).
	Severity string `json:"severity"`
}

// Engine produces diagnostic explanations from resolver results, confidence, and TTL data.
type Engine struct{}

// NewEngine creates a new diagnostics engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Diagnose analyses the current state and produces a diagnosis.
func (e *Engine) Diagnose(results []resolver.Result, score confidence.Score, ttlReport ttl.Report) Diagnosis {
	// Scenario 1: No resolvers found the record
	if score.AuthFound == 0 && score.PublicFound == 0 {
		return e.noRecordFound(results, score)
	}

	// Scenario 2: Auth found it but public hasn't propagated yet
	if score.AuthFound > 0 && score.PublicFound == 0 {
		return e.authOnlyPropagation(score, ttlReport)
	}

	// Scenario 3: Partial propagation
	if score.Overall < 100 && score.Overall > 0 {
		return e.partialPropagation(results, score, ttlReport)
	}

	// Scenario 4: Full propagation
	return Diagnosis{
		Scenario:    "full_propagation",
		Summary:     "Record fully propagated across all resolvers",
		Explanation: "All queried resolvers returned the expected TXT record value.",
		Severity:    "info",
	}
}

func (e *Engine) noRecordFound(results []resolver.Result, score confidence.Score) Diagnosis {
	var errorResolvers []string
	for _, r := range results {
		if r.Error != "" {
			errorResolvers = append(errorResolvers, fmt.Sprintf("%s (%s)", r.Resolver, r.Error))
		}
	}

	explanation := "No resolver returned the expected TXT record. "
	if len(errorResolvers) > 0 {
		explanation += fmt.Sprintf("%d resolver(s) returned errors: %s. ",
			len(errorResolvers), strings.Join(errorResolvers, "; "))
	}
	explanation += "The record may not have been created yet, or the domain name may be incorrect."

	return Diagnosis{
		Scenario:    "no_record",
		Summary:     "TXT record not found by any resolver",
		Explanation: explanation,
		Severity:    "error",
		Recommendations: []string{
			"Verify the TXT record has been created at your DNS provider",
			"Check that the domain name is correct (including _acme-challenge. prefix)",
			"Ensure the record value matches exactly what your ACME client expects",
			"Wait a few seconds and try again — creation may still be in progress",
		},
	}
}

func (e *Engine) authOnlyPropagation(score confidence.Score, ttlReport ttl.Report) Diagnosis {
	explanation := fmt.Sprintf(
		"Authoritative nameservers have the record (%d/%d found), but no public resolver has picked it up yet. ",
		score.AuthFound, score.AuthTotal,
	)
	if ttlReport.MaxTTL > 0 {
		explanation += fmt.Sprintf(
			"Based on TTL values (max %ds), full propagation is expected within %s.",
			ttlReport.MaxTTL, ttlReport.EstimatedFullPropagation,
		)
	} else {
		explanation += "Propagation is in progress — public resolvers will pick up the record as their caches expire."
	}

	return Diagnosis{
		Scenario:    "auth_only",
		Summary:     "Record found on authoritative NS but not yet on public resolvers",
		Explanation: explanation,
		Severity:    "warning",
		Recommendations: []string{
			"Wait for DNS caches to expire — this is normal propagation behaviour",
			"Consider using a lower TTL for ACME challenge records",
		},
	}
}

func (e *Engine) partialPropagation(results []resolver.Result, score confidence.Score, ttlReport ttl.Report) Diagnosis {
	var missingResolvers []string
	for _, r := range results {
		if r.Error == "" && !r.Found {
			missingResolvers = append(missingResolvers, r.Resolver)
		}
	}

	explanation := fmt.Sprintf(
		"Record is partially propagated (confidence: %.1f%%). Auth: %d/%d, Public: %d/%d. ",
		score.Overall, score.AuthFound, score.AuthTotal, score.PublicFound, score.PublicTotal,
	)
	if len(missingResolvers) > 0 {
		explanation += fmt.Sprintf("Not yet visible on: %s. ", strings.Join(missingResolvers, ", "))
	}

	return Diagnosis{
		Scenario:    "partial_propagation",
		Summary:     fmt.Sprintf("Partial propagation — %.1f%% confidence", score.Overall),
		Explanation: explanation,
		Severity:    "warning",
		Recommendations: []string{
			"Continue waiting — propagation is in progress",
			"The record should reach full propagation soon",
		},
	}
}

