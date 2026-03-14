package output

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/axelspire/at3am/internal/confidence"
)

func TestMetrics_Update(t *testing.T) {
	m := NewMetrics()
	status := PollStatus{
		Domain:  "example.com",
		Attempt: 5,
		Score: confidence.Score{
			Overall: 85.0, AuthFound: 2, AuthTotal: 2,
			PublicFound: 15, PublicTotal: 18,
		},
		ConsecPasses: 1,
		Ready:        false,
	}
	m.Update(status)

	if m.pollCount != 5 {
		t.Errorf("expected pollCount 5, got %d", m.pollCount)
	}
	if m.currentConfidence != 85.0 {
		t.Errorf("expected confidence 85.0, got %.1f", m.currentConfidence)
	}
	if m.domain != "example.com" {
		t.Errorf("expected domain example.com, got %s", m.domain)
	}
}

func TestMetrics_Handler(t *testing.T) {
	m := NewMetrics()
	m.Update(PollStatus{
		Domain:  "test.com",
		Attempt: 3,
		Score: confidence.Score{
			Overall: 95.0, AuthFound: 2, AuthTotal: 2,
			PublicFound: 17, PublicTotal: 18,
		},
		ConsecPasses: 2,
		Ready:        true,
	})

	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body := rr.Body.String()

	checks := []string{
		"at3am_poll_count",
		"at3am_confidence",
		"at3am_consecutive_passes",
		"at3am_ready",
		"at3am_auth_found",
		"at3am_auth_total",
		"at3am_public_found",
		"at3am_public_total",
		`domain="test.com"`,
	}

	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Errorf("expected %q in metrics output, body:\n%s", check, body)
		}
	}

	// Check content type
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("expected text/plain content type, got %s", ct)
	}

	// Ready should be 1
	if !strings.Contains(body, "at3am_ready{domain=\"test.com\"} 1") {
		t.Error("expected ready=1")
	}
}

func TestMetrics_Handler_NotReady(t *testing.T) {
	m := NewMetrics()
	m.Update(PollStatus{Ready: false})

	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !strings.Contains(rr.Body.String(), "at3am_ready{domain=\"\"} 0") {
		t.Error("expected ready=0")
	}
}

func TestStartMetricsServer(t *testing.T) {
	m := NewMetrics()
	m.Update(PollStatus{Domain: "srv-test.com", Attempt: 1})

	srv, err := StartMetricsServer(0, m) // port 0 = random available port
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer srv.Close()

	// The server starts on port 0 which won't be easily accessible,
	// but we can verify the server was created
	if srv == nil {
		t.Error("expected non-nil server")
	}
}

func TestStartMetricsServer_Integration(t *testing.T) {
	m := NewMetrics()
	m.Update(PollStatus{Domain: "integ.com", Attempt: 2, Score: confidence.Score{Overall: 50}})

	// Use httptest instead for a proper integration test
	mux := http.NewServeMux()
	mux.Handle("/metrics", m.Handler())
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatalf("failed to GET /metrics: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "integ.com") {
		t.Error("expected domain in metrics")
	}
}

