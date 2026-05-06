package installer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchNodeIndex(t *testing.T) {
	payload := []map[string]any{
		{"version": "v24.1.0", "lts": false},
		{"version": "v22.14.0", "lts": "Jod"},
		{"version": "v20.19.2", "lts": "Iron"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	orig := httpClient
	httpClient = &http.Client{Transport: &hostRewriteTransport{target: srv.URL}}
	defer func() { httpClient = orig }()

	releases, err := FetchNodeIndex()
	if err != nil {
		t.Fatalf("FetchNodeIndex() error: %v", err)
	}

	if len(releases) != 3 {
		t.Fatalf("expected 3 releases, got %d", len(releases))
	}

	// "v" prefix must be stripped.
	if releases[0].Version != "24.1.0" {
		t.Errorf("version[0] = %q, want 24.1.0", releases[0].Version)
	}
	if releases[1].Version != "22.14.0" {
		t.Errorf("version[1] = %q, want 22.14.0", releases[1].Version)
	}

	// LTS field should round-trip as string for LTS releases.
	lts, ok := releases[1].LTS.(string)
	if !ok || lts != "Jod" {
		t.Errorf("releases[1].LTS = %v, want \"Jod\"", releases[1].LTS)
	}
	if releases[0].LTS != false {
		t.Errorf("releases[0].LTS = %v, want false", releases[0].LTS)
	}
}

// hostRewriteTransport redirects all requests to a fixed base URL (test server).
type hostRewriteTransport struct {
	target string // e.g. "http://127.0.0.1:PORT"
}

func (t *hostRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned, err := http.NewRequest(req.Method, t.target+req.URL.Path, req.Body)
	if err != nil {
		return nil, err
	}
	cloned.Header = req.Header
	return http.DefaultTransport.RoundTrip(cloned)
}
