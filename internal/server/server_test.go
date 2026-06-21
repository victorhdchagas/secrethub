package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeSetupRedirect(t *testing.T) {
	t.Parallel()

	s := New(Config{Host: "127.0.0.1", Port: 4949, DataDir: t.TempDir()})
	server := httptest.NewServer(s)
	defer server.Close()

	client := &http.Client{CheckRedirect: func(r *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	resp, err := client.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302, got %d", resp.StatusCode)
	}
}

func TestServeSetupPage(t *testing.T) {
	t.Parallel()

	s := New(Config{Host: "127.0.0.1", Port: 4949, DataDir: t.TempDir()})
	server := httptest.NewServer(s)
	defer server.Close()

	resp, err := http.Get(server.URL + "/setup")
	if err != nil {
		t.Fatalf("GET /setup: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}
