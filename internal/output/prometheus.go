package output

import (
	"fmt"
	"net/http"
	"sync"
)

// Metrics holds Prometheus-style metrics for at3am.
type Metrics struct {
	mu               sync.RWMutex
	pollCount        int
	currentConfidence float64
	consecutivePasses int
	domain           string
	ready            bool
	authFound        int
	authTotal        int
	publicFound      int
	publicTotal      int
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Update updates the metrics with the latest poll status.
func (m *Metrics) Update(status PollStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pollCount = status.Attempt
	m.currentConfidence = status.Score.Overall
	m.consecutivePasses = status.ConsecPasses
	m.domain = status.Domain
	m.ready = status.Ready
	m.authFound = status.Score.AuthFound
	m.authTotal = status.Score.AuthTotal
	m.publicFound = status.Score.PublicFound
	m.publicTotal = status.Score.PublicTotal
}

// Handler returns an HTTP handler that serves Prometheus metrics.
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		defer m.mu.RUnlock()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		readyVal := 0
		if m.ready {
			readyVal = 1
		}

		fmt.Fprintf(w, "# HELP at3am_poll_count Total number of poll cycles.\n")
		fmt.Fprintf(w, "# TYPE at3am_poll_count counter\n")
		fmt.Fprintf(w, "at3am_poll_count{domain=%q} %d\n", m.domain, m.pollCount)
		fmt.Fprintf(w, "# HELP at3am_confidence Current confidence score.\n")
		fmt.Fprintf(w, "# TYPE at3am_confidence gauge\n")
		fmt.Fprintf(w, "at3am_confidence{domain=%q} %.2f\n", m.domain, m.currentConfidence)
		fmt.Fprintf(w, "# HELP at3am_consecutive_passes Current consecutive passes above threshold.\n")
		fmt.Fprintf(w, "# TYPE at3am_consecutive_passes gauge\n")
		fmt.Fprintf(w, "at3am_consecutive_passes{domain=%q} %d\n", m.domain, m.consecutivePasses)
		fmt.Fprintf(w, "# HELP at3am_ready Whether propagation is confirmed.\n")
		fmt.Fprintf(w, "# TYPE at3am_ready gauge\n")
		fmt.Fprintf(w, "at3am_ready{domain=%q} %d\n", m.domain, readyVal)
		fmt.Fprintf(w, "# HELP at3am_auth_found Authoritative resolvers that found the record.\n")
		fmt.Fprintf(w, "# TYPE at3am_auth_found gauge\n")
		fmt.Fprintf(w, "at3am_auth_found{domain=%q} %d\n", m.domain, m.authFound)
		fmt.Fprintf(w, "# HELP at3am_auth_total Total authoritative resolvers queried.\n")
		fmt.Fprintf(w, "# TYPE at3am_auth_total gauge\n")
		fmt.Fprintf(w, "at3am_auth_total{domain=%q} %d\n", m.domain, m.authTotal)
		fmt.Fprintf(w, "# HELP at3am_public_found Public resolvers that found the record.\n")
		fmt.Fprintf(w, "# TYPE at3am_public_found gauge\n")
		fmt.Fprintf(w, "at3am_public_found{domain=%q} %d\n", m.domain, m.publicFound)
		fmt.Fprintf(w, "# HELP at3am_public_total Total public resolvers queried.\n")
		fmt.Fprintf(w, "# TYPE at3am_public_total gauge\n")
		fmt.Fprintf(w, "at3am_public_total{domain=%q} %d\n", m.domain, m.publicTotal)
	})
}

// StartServer starts a Prometheus metrics server on the given port.
// Returns a function to stop the server.
func StartMetricsServer(port int, metrics *Metrics) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Log error but don't crash
			fmt.Printf("metrics server error: %v\n", err)
		}
	}()

	return srv, nil
}

