package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeHealth(t *testing.T) {
	t.Parallel()

	s := New(Config{Host: "127.0.0.1", Port: 4949})
	server := httptest.NewServer(s.mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}
