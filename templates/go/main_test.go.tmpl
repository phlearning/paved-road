package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(t *testing.T, srv *httptest.Server, path string) (int, string) {
	t.Helper()
	resp, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestRoutes(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	for _, path := range []string{"/", "/healthz", "/readyz"} {
		if status, _ := get(t, srv, path); status != http.StatusOK {
			t.Errorf("GET %s: status %d", path, status)
		}
	}
	if status, _ := get(t, srv, "/missing"); status != http.StatusNotFound {
		t.Errorf("GET /missing: status %d, want 404", status)
	}
}

func TestMetricsTrackRoutesNotProbes(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	get(t, srv, "/")
	get(t, srv, "/healthz")
	_, body := get(t, srv, "/metrics")

	if !strings.Contains(body, `http_requests_total{handler="/",method="GET",status="2xx"}`) {
		t.Errorf("missing request counter for /:\n%s", body)
	}
	if strings.Contains(body, `handler="/healthz"`) {
		t.Error("probes must not be tracked")
	}
}
