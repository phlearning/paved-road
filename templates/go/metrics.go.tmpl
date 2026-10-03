package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTP metrics with the names and labels the platform dashboard expects.
var (
	requests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests handled.",
	}, []string{"method", "handler", "status"})

	latency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "HTTP request latency.",
	}, []string{"method", "handler"})
)

// handle registers a business route and records its metrics under the route
// pattern, which keeps label cardinality bounded.
func handle(mux *http.ServeMux, pattern string, h http.HandlerFunc) {
	// "GET /items/{id}" is labelled "/items/{id}", "GET /{$}" is labelled "/".
	_, path, _ := strings.Cut(pattern, " ")
	handler := strings.TrimSuffix(path, "{$}")
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		h(rec, r)
		requests.WithLabelValues(r.Method, handler, strconv.Itoa(rec.status/100)+"xx").Inc()
		latency.WithLabelValues(r.Method, handler).Observe(time.Since(start).Seconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func metricsHandler() http.Handler {
	return promhttp.Handler()
}
