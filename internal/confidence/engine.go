// Package confidence implements the MPIC-aware confidence scoring engine.
package confidence

import (
	"github.com/axelspire/at3am/internal/resolver"
)

// authMode controls how the required number of auth resolvers is derived.
type authMode int

const (
	authModeFixed  authMode = iota // exact count stored in authCount
	authModeAll                    // ALL responding auth resolvers
	authModeCeilN2                 // ceil(N/2) of responding auth resolvers
)

// publicMode controls how the required number of public resolvers is derived.
type publicMode int

const (
	publicModeFixed     publicMode = iota // exact count stored in publicCount
	publicModeAllMinus2                   // all public resolvers minus 2
)

// ProfileType represents the different readiness profiles.
type ProfileType int

const (
	ProfileStrict  ProfileType = iota
	ProfileDefault
	ProfileFast
	ProfileYolo
)

// ProfileFromString converts a string profile name to ProfileType.
func ProfileFromString(profile string) ProfileType {
	switch profile {
	case "strict":
		return ProfileStrict
	case "default":
		return ProfileDefault
	case "fast":
		return ProfileFast
	case "yolo":
		return ProfileYolo
	default:
		return ProfileDefault
	}
}

// Score represents the confidence score for a single poll pass.
type Score struct {
	// AuthTotal is the total number of authoritative resolvers queried.
	AuthTotal int `json:"auth_total"`
	// AuthFound is the number of authoritative resolvers that found the record.
	AuthFound int `json:"auth_found"`
	// AuthErrors is the number of authoritative resolvers that returned errors.
	AuthErrors int `json:"auth_errors"`
	// PublicTotal is the total number of public resolvers queried.
	PublicTotal int `json:"public_total"`
	// PublicFound is the number of public resolvers that found the record.
	PublicFound int `json:"public_found"`
	// PublicErrors is the number of public resolvers that returned errors.
	PublicErrors int `json:"public_errors"`
	// AuthScore is the authoritative NS score (0-100), for display purposes.
	AuthScore float64 `json:"auth_score"`
	// PublicScore is the public resolver score (0-100), for display purposes.
	PublicScore float64 `json:"public_score"`
	// Overall weighted confidence (0-100), for display purposes.
	Overall float64 `json:"overall"`
	// AuthRequired is the minimum auth resolvers that must confirm for Ready.
	AuthRequired int `json:"auth_required"`
	// PublicRequired is the minimum public resolvers that must confirm for Ready.
	PublicRequired int `json:"public_required"`
	// Ready is true when auth_correct >= auth_threshold AND public_correct >= public_threshold.
	Ready bool `json:"ready"`
	// AboveThreshold mirrors Ready for backward compatibility.
	AboveThreshold bool `json:"above_threshold"`

	// DNSSEC aggregates — only populated when --dnssec-validate is set.
	// DNSSECChecked is true if at least one resolver had the DO bit set on its query.
	DNSSECChecked bool `json:"dnssec_checked,omitempty"`
	// DNSSECValidCount is the number of resolvers that returned the AD (Authenticated Data) bit.
	DNSSECValidCount int `json:"dnssec_valid_count,omitempty"`
	// DNSSECTotal is the number of resolvers that returned a result (and were asked to validate).
	DNSSECTotal int `json:"dnssec_total,omitempty"`
}

// Engine computes confidence scores from resolver results.
type Engine struct {
	// Display weights — used only for the Overall percentage shown in output.
	authWeight   float64
	publicWeight float64

	// Profile-based readiness thresholds.
	hasProfile  bool
	aMode       authMode
	authCount   int // used when aMode == authModeFixed
	pMode       publicMode
	publicCount int // used when pMode == publicModeFixed

	// Legacy percentage threshold (used when hasProfile == false).
	threshold float64
}

// NewEngine creates a legacy engine that signals Ready when overall >= threshold.
func NewEngine(authWeight, publicWeight, threshold float64) *Engine {
	return &Engine{
		authWeight:   authWeight,
		publicWeight: publicWeight,
		threshold:    threshold,
		hasProfile:   false,
	}
}

// NewEngineWithProfile creates an engine using the profile readiness rules:
//
//	strict  — ALL auth AND (ALL−2) public
//	default — ALL auth AND ≥1 public
//	fast    — ≥ceil(N/2) auth AND ≥1 public
//	yolo    — ≥1 auth AND ≥0 public (auth alone is enough)
func NewEngineWithProfile(profile ProfileType) *Engine {
	e := &Engine{
		hasProfile:   true,
		authWeight:   0.6,
		publicWeight: 0.4,
	}
	switch profile {
	case ProfileStrict:
		e.aMode, e.pMode = authModeAll, publicModeAllMinus2
		e.threshold = 100.0
	case ProfileDefault:
		e.aMode, e.pMode, e.publicCount = authModeAll, publicModeFixed, 1
		e.threshold = 95.0
	case ProfileFast:
		e.aMode, e.pMode, e.publicCount = authModeCeilN2, publicModeFixed, 1
		e.threshold = 80.0
	case ProfileYolo:
		e.aMode, e.authCount = authModeFixed, 1
		e.pMode, e.publicCount = publicModeFixed, 0
		e.threshold = 50.0
	}
	return e
}

// Calculate computes a Score from resolver results, checking for the expected value.
func (e *Engine) Calculate(results []resolver.Result, expected string) Score {
	var s Score

	for _, r := range results {
		if r.AuthoritativeNS {
			s.AuthTotal++
			if r.Error != "" {
				s.AuthErrors++
				continue
			}
			if containsValue(r.Values, expected) {
				s.AuthFound++
			}
		} else {
			s.PublicTotal++
			if r.Error != "" {
				s.PublicErrors++
				continue
			}
			if containsValue(r.Values, expected) {
				s.PublicFound++
			}
		}
		// Aggregate DNSSEC status across all responding resolvers.
		if r.DNSSECChecked {
			s.DNSSECChecked = true
			s.DNSSECTotal++
			if r.DNSSECValid {
				s.DNSSECValidCount++
			}
		}
	}

	// Per-category percentages — used for display and legacy threshold check.
	respondingAuth := s.AuthTotal - s.AuthErrors
	if respondingAuth > 0 {
		s.AuthScore = float64(s.AuthFound) / float64(respondingAuth) * 100
	}
	respondingPublic := s.PublicTotal - s.PublicErrors
	if respondingPublic > 0 {
		s.PublicScore = float64(s.PublicFound) / float64(respondingPublic) * 100
	}

	// Weighted overall for display.
	switch {
	case s.AuthTotal > 0 && s.PublicTotal > 0:
		s.Overall = e.authWeight*s.AuthScore + e.publicWeight*s.PublicScore
	case s.AuthTotal > 0:
		s.Overall = s.AuthScore
	case s.PublicTotal > 0:
		s.Overall = s.PublicScore
	}

	// Derive required counts and readiness.
	s.AuthRequired = e.requiredAuth(respondingAuth)
	s.PublicRequired = e.requiredPublic(respondingPublic)

	if e.hasProfile {
		s.Ready = s.AuthFound >= s.AuthRequired && s.PublicFound >= s.PublicRequired
	} else {
		// Legacy mode: Ready when the weighted overall percentage clears the threshold.
		s.Ready = s.Overall >= e.threshold
	}

	s.AboveThreshold = s.Ready
	return s
}

// requiredAuth returns the minimum number of auth resolvers that must confirm.
func (e *Engine) requiredAuth(responding int) int {
	switch e.aMode {
	case authModeAll:
		return responding
	case authModeCeilN2:
		return (responding + 1) / 2
	default: // authModeFixed
		return e.authCount
	}
}

// requiredPublic returns the minimum number of public resolvers that must confirm.
func (e *Engine) requiredPublic(responding int) int {
	switch e.pMode {
	case publicModeAllMinus2:
		if responding <= 2 {
			return responding
		}
		return responding - 2
	default: // publicModeFixed
		return e.publicCount
	}
}

func containsValue(values []string, expected string) bool {
	for _, v := range values {
		// Direct match
		if v == expected {
			return true
		}
		// Match with quotes stripped (some DNS providers store TXT with quotes)
		if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
			unquoted := v[1 : len(v)-1]
			if unquoted == expected {
				return true
			}
		}
	}
	return false
}

