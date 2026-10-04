package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
)

const serviceName = "hello"

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	handle(mux, "GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"service": serviceName, "message": "Hello from the paved road"})
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ready"})
	})
	mux.Handle("GET /metrics", metricsHandler())
	return mux
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	slog.Info("listening", "service", serviceName, "port", port)
	if err := http.ListenAndServe(":"+port, newMux()); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
