package confidence

import (
	"math"
	"testing"
	"time"

	"github.com/axelspire/at3am/internal/resolver"
)

func TestCalculate_AllFound(t *testing.T) {
	e := NewEngine(0.6, 0.4, 95.0)
	results := []resolver.Result{
		{Resolver: "10.0.0.1", Found: true, Values: []string{"token"}, AuthoritativeNS: true, Latency: time.Millisecond},
		{Resolver: "10.0.0.2", Found: true, Values: []string{"token"}, AuthoritativeNS: true, Latency: time.Millisecond},
		{Resolver: "8.8.8.8", Found: true, Values: []string{"token"}, AuthoritativeNS: false, Latency: time.Millisecond},
		{Resolver: "1.1.1.1", Found: true, Values: []string{"token"}, AuthoritativeNS: false, Latency: time.Millisecond},
	}
	score := e.Calculate(results, "token")
	if score.Overall != 100.0 {
		t.Errorf("expected 100.0, got %.1f", score.Overall)
	}
	if !score.AboveThreshold {
		t.Error("expected above threshold")
	}
	if score.AuthFound != 2 || score.PublicFound != 2 {
		t.Errorf("expected auth=2, public=2, got auth=%d, public=%d", score.AuthFound, score.PublicFound)
	}
}

func TestCalculate_NoneFound(t *testing.T) {
	e := NewEngine(0.6, 0.4, 95.0)
	results := []resolver.Result{
		{Resolver: "10.0.0.1", Found: false, AuthoritativeNS: true},
		{Resolver: "8.8.8.8", Found: false, AuthoritativeNS: false},
	}
	score := e.Calculate(results, "token")
	if score.Overall != 0.0 {
		t.Errorf("expected 0.0, got %.1f", score.Overall)
	}
	if score.AboveThreshold {
		t.Error("expected below threshold")
	}
}

func TestCalculate_AuthOnlyFound(t *testing.T) {
	e := NewEngine(0.6, 0.4, 95.0)
	results := []resolver.Result{
		{Resolver: "10.0.0.1", Found: true, Values: []string{"token"}, AuthoritativeNS: true},
		{Resolver: "10.0.0.2", Found: true, Values: []string{"token"}, AuthoritativeNS: true},
		{Resolver: "8.8.8.8", Found: false, AuthoritativeNS: false},
		{Resolver: "1.1.1.1", Found: false, AuthoritativeNS: false},
	}
	score := e.Calculate(results, "token")
	// Auth = 100%, Public = 0%, Overall = 0.6*100 + 0.4*0 = 60
	if score.Overall != 60.0 {
		t.Errorf("expected 60.0, got %.1f", score.Overall)
	}
}

func TestCalculate_PartialPublic(t *testing.T) {
	e := NewEngine(0.6, 0.4, 95.0)
	results := []resolver.Result{
		{Resolver: "10.0.0.1", Found: true, Values: []string{"token"}, AuthoritativeNS: true},
		{Resolver: "8.8.8.8", Found: true, Values: []string{"token"}, AuthoritativeNS: false},
		{Resolver: "1.1.1.1", Found: false, AuthoritativeNS: false},
	}
	score := e.Calculate(results, "token")
	// Auth = 100%, Public = 50%, Overall = 0.6*100 + 0.4*50 = 80
	if score.Overall != 80.0 {
		t.Errorf("expected 80.0, got %.1f", score.Overall)
	}
}

func TestCalculate_WithErrors(t *testing.T) {
	e := NewEngine(0.6, 0.4, 95.0)
	results := []resolver.Result{
		{Resolver: "10.0.0.1", Found: true, Values: []string{"token"}, AuthoritativeNS: true},
		{Resolver: "10.0.0.2", Error: "timeout", AuthoritativeNS: true},
		{Resolver: "8.8.8.8", Found: true, Values: []string{"token"}, AuthoritativeNS: false},
		{Resolver: "1.1.1.1", Error: "timeout", AuthoritativeNS: false},
	}
	score := e.Calculate(results, "token")
	// Auth: 1 responding, 1 found = 100%. Public: 1 responding, 1 found = 100%
	if score.Overall != 100.0 {
		t.Errorf("expected 100.0, got %.1f", score.Overall)
	}
	if score.AuthErrors != 1 || score.PublicErrors != 1 {
		t.Errorf("expected 1 auth error and 1 public error")
	}
}

func TestCalculate_WrongValue(t *testing.T) {
	e := NewEngine(0.6, 0.4, 95.0)
	results := []resolver.Result{
		{Resolver: "10.0.0.1", Found: true, Values: []string{"wrong-token"}, AuthoritativeNS: true},
		{Resolver: "8.8.8.8", Found: true, Values: []string{"wrong-token"}, AuthoritativeNS: false},
	}
	score := e.Calculate(results, "expected-token")
	if score.Overall != 0.0 {
		t.Errorf("expected 0.0, got %.1f", score.Overall)
	}
}

func TestCalculate_OnlyAuth(t *testing.T) {
	e := NewEngine(0.6, 0.4, 95.0)
	results := []resolver.Result{
		{Resolver: "10.0.0.1", Found: true, Values: []string{"token"}, AuthoritativeNS: true},
	}
	score := e.Calculate(results, "token")
	// Only auth present, should use auth score directly
	if score.Overall != 100.0 {
		t.Errorf("expected 100.0, got %.1f", score.Overall)
	}
}

func TestCalculate_OnlyPublic(t *testing.T) {
	e := NewEngine(0.6, 0.4, 95.0)
	results := []resolver.Result{
		{Resolver: "8.8.8.8", Found: true, Values: []string{"token"}, AuthoritativeNS: false},
	}
	score := e.Calculate(results, "token")
	if score.Overall != 100.0 {
		t.Errorf("expected 100.0, got %.1f", score.Overall)
	}
}

func TestCalculate_Empty(t *testing.T) {
	e := NewEngine(0.6, 0.4, 95.0)
	score := e.Calculate(nil, "token")
	if score.Overall != 0.0 {
		t.Errorf("expected 0.0, got %.1f", score.Overall)
	}
}

func TestCalculate_AllErrors(t *testing.T) {
	e := NewEngine(0.6, 0.4, 95.0)
	results := []resolver.Result{
		{Resolver: "10.0.0.1", Error: "timeout", AuthoritativeNS: true},
		{Resolver: "8.8.8.8", Error: "timeout", AuthoritativeNS: false},
	}
	score := e.Calculate(results, "token")
	if score.Overall != 0.0 {
		t.Errorf("expected 0.0, got %.1f", score.Overall)
	}
}

// helpers

func authResults(n, found int) []resolver.Result {
	r := make([]resolver.Result, n)
	for i := range r {
		r[i] = resolver.Result{Resolver: "auth", AuthoritativeNS: true}
		if i < found {
			r[i].Found = true
			r[i].Values = []string{"token"}
		}
	}
	return r
}

func pubResults(n, found int) []resolver.Result {
	r := make([]resolver.Result, n)
	for i := range r {
		r[i] = resolver.Result{Resolver: "pub", AuthoritativeNS: false}
		if i < found {
			r[i].Found = true
			r[i].Values = []string{"token"}
		}
	}
	return r
}

func concat(a, b []resolver.Result) []resolver.Result {
	return append(a, b...)
}

// Profile readiness tests — READY = (auth_correct >= auth_threshold) AND (public_correct >= public_threshold)

func TestProfile_Strict_Ready(t *testing.T) {
	// strict: ALL auth AND (ALL−2) public
	e := NewEngineWithProfile(ProfileStrict)
	// 4 auth all found, 6 public with 4 found (6-2=4 required) → READY
	results := concat(authResults(4, 4), pubResults(6, 4))
	s := e.Calculate(results, "token")
	if !s.Ready {
		t.Errorf("strict: expected Ready with all auth and all-2 public; auth=%d/%d pub=%d/%d (required auth=%d pub=%d)",
			s.AuthFound, s.AuthTotal, s.PublicFound, s.PublicTotal, s.AuthRequired, s.PublicRequired)
	}
}

func TestProfile_Strict_NotReady_MissingAuth(t *testing.T) {
	// strict: one auth missing → NOT READY
	e := NewEngineWithProfile(ProfileStrict)
	results := concat(authResults(4, 3), pubResults(6, 4))
	s := e.Calculate(results, "token")
	if s.Ready {
		t.Errorf("strict: expected NOT Ready when one auth missing; auth=%d/%d pub=%d/%d (required auth=%d pub=%d)",
			s.AuthFound, s.AuthTotal, s.PublicFound, s.PublicTotal, s.AuthRequired, s.PublicRequired)
	}
}

func TestProfile_Strict_NotReady_MissingPublic(t *testing.T) {
	// strict: public found = ALL-3 (one short of ALL-2) → NOT READY
	e := NewEngineWithProfile(ProfileStrict)
	results := concat(authResults(4, 4), pubResults(6, 3))
	s := e.Calculate(results, "token")
	if s.Ready {
		t.Errorf("strict: expected NOT Ready when public < all-2; auth=%d/%d pub=%d/%d (required auth=%d pub=%d)",
			s.AuthFound, s.AuthTotal, s.PublicFound, s.PublicTotal, s.AuthRequired, s.PublicRequired)
	}
}

func TestProfile_Default_Ready(t *testing.T) {
	// default: ALL auth AND ≥1 public
	e := NewEngineWithProfile(ProfileDefault)
	results := concat(authResults(6, 6), pubResults(18, 1))
	s := e.Calculate(results, "token")
	if !s.Ready {
		t.Errorf("default: expected Ready with all auth and 1 public; auth=%d/%d pub=%d/%d (required auth=%d pub=%d)",
			s.AuthFound, s.AuthTotal, s.PublicFound, s.PublicTotal, s.AuthRequired, s.PublicRequired)
	}
}

func TestProfile_Default_NotReady_ZeroPublic(t *testing.T) {
	// default: all auth but 0 public found → NOT READY (public gate not cleared)
	e := NewEngineWithProfile(ProfileDefault)
	results := concat(authResults(6, 6), pubResults(18, 0))
	s := e.Calculate(results, "token")
	if s.Ready {
		t.Errorf("default: expected NOT Ready when 0 public found; auth=%d/%d pub=%d/%d (required auth=%d pub=%d)",
			s.AuthFound, s.AuthTotal, s.PublicFound, s.PublicTotal, s.AuthRequired, s.PublicRequired)
	}
}

func TestProfile_Default_NotReady_MissingOneAuth(t *testing.T) {
	// default: 5/6 auth, 1 public → NOT READY (auth must be ALL)
	e := NewEngineWithProfile(ProfileDefault)
	results := concat(authResults(6, 5), pubResults(18, 1))
	s := e.Calculate(results, "token")
	if s.Ready {
		t.Errorf("default: expected NOT Ready when one auth missing; auth=%d/%d pub=%d/%d (required auth=%d pub=%d)",
			s.AuthFound, s.AuthTotal, s.PublicFound, s.PublicTotal, s.AuthRequired, s.PublicRequired)
	}
}

func TestProfile_Fast_Ready(t *testing.T) {
	// fast: ≥ceil(N/2) auth AND ≥1 public
	// 6 auth resolvers → ceil(6/2)=3 required
	e := NewEngineWithProfile(ProfileFast)
	results := concat(authResults(6, 3), pubResults(18, 1))
	s := e.Calculate(results, "token")
	if !s.Ready {
		t.Errorf("fast: expected Ready with ceil(N/2) auth and 1 public; auth=%d/%d pub=%d/%d (required auth=%d pub=%d)",
			s.AuthFound, s.AuthTotal, s.PublicFound, s.PublicTotal, s.AuthRequired, s.PublicRequired)
	}
}

func TestProfile_Fast_NotReady_BelowCeil(t *testing.T) {
	// fast: 2/6 auth found (ceil(6/2)=3 required) → NOT READY
	e := NewEngineWithProfile(ProfileFast)
	results := concat(authResults(6, 2), pubResults(18, 1))
	s := e.Calculate(results, "token")
	if s.Ready {
		t.Errorf("fast: expected NOT Ready when auth < ceil(N/2); auth=%d/%d pub=%d/%d (required auth=%d pub=%d)",
			s.AuthFound, s.AuthTotal, s.PublicFound, s.PublicTotal, s.AuthRequired, s.PublicRequired)
	}
}

func TestProfile_Fast_OddAuthCount(t *testing.T) {
	// fast: 5 auth → ceil(5/2)=3, so 3 found → READY
	e := NewEngineWithProfile(ProfileFast)
	results := concat(authResults(5, 3), pubResults(18, 1))
	s := e.Calculate(results, "token")
	if !s.Ready {
		t.Errorf("fast: expected Ready with ceil(5/2)=3 auth found; auth=%d/%d pub=%d/%d (required auth=%d pub=%d)",
			s.AuthFound, s.AuthTotal, s.PublicFound, s.PublicTotal, s.AuthRequired, s.PublicRequired)
	}
}

func TestProfile_Yolo_Ready(t *testing.T) {
	// yolo: ≥1 auth AND ≥0 public (auth alone is enough)
	e := NewEngineWithProfile(ProfileYolo)
	results := authResults(6, 1)
	s := e.Calculate(results, "token")
	if !s.Ready {
		t.Errorf("yolo: expected Ready with just 1 auth; auth=%d/%d pub=%d/%d (required auth=%d pub=%d)",
			s.AuthFound, s.AuthTotal, s.PublicFound, s.PublicTotal, s.AuthRequired, s.PublicRequired)
	}
}

func TestProfile_Yolo_NotReady_ZeroAuth(t *testing.T) {
	// yolo: 0 auth found → NOT READY (still need ≥1)
	e := NewEngineWithProfile(ProfileYolo)
	results := concat(authResults(6, 0), pubResults(18, 18))
	s := e.Calculate(results, "token")
	if s.Ready {
		t.Errorf("yolo: expected NOT Ready with 0 auth even with all public; auth=%d/%d pub=%d/%d (required auth=%d pub=%d)",
			s.AuthFound, s.AuthTotal, s.PublicFound, s.PublicTotal, s.AuthRequired, s.PublicRequired)
	}
}

func TestProfile_AuthRequired_PublicRequired_Populated(t *testing.T) {
	// Verify the AuthRequired/PublicRequired fields are correctly populated in the Score
	e := NewEngineWithProfile(ProfileDefault)
	results := concat(authResults(6, 6), pubResults(18, 5))
	s := e.Calculate(results, "token")
	if s.AuthRequired != 6 {
		t.Errorf("default: AuthRequired should be 6 (ALL), got %d", s.AuthRequired)
	}
	if s.PublicRequired != 1 {
		t.Errorf("default: PublicRequired should be 1, got %d", s.PublicRequired)
	}
}

func TestContainsValue(t *testing.T) {
	if !containsValue([]string{"a", "b", "c"}, "b") {
		t.Error("expected true")
	}
	if containsValue([]string{"a", "b"}, "c") {
		t.Error("expected false")
	}
	if containsValue(nil, "a") {
		t.Error("expected false for nil")
	}
}

func TestCalculate_ThresholdBoundary(t *testing.T) {
	e := NewEngine(0.6, 0.4, 95.0)
	// Construct result that gives exactly 95%
	results := []resolver.Result{
		{Resolver: "10.0.0.1", Found: true, Values: []string{"t"}, AuthoritativeNS: true},
	}
	// 19 public found, 1 not
	for i := 0; i < 19; i++ {
		results = append(results, resolver.Result{Resolver: "pub", Found: true, Values: []string{"t"}, AuthoritativeNS: false})
	}
	results = append(results, resolver.Result{Resolver: "pub-miss", Found: false, AuthoritativeNS: false})

	score := e.Calculate(results, "t")
	expected := 0.6*100 + 0.4*95.0
	if math.Abs(score.Overall-expected) > 0.1 {
		t.Errorf("expected ~%.1f, got %.1f", expected, score.Overall)
	}
}

