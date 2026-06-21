package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeLoginPage(t *testing.T) {
	t.Parallel()

	s := New(Config{Host: "127.0.0.1", Port: 4949, DataDir: t.TempDir()})
	server := httptest.NewServer(s.mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}
