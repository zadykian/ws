package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientGivesUpOnAServerThatNeverAnswers(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	defer srv.Close()
	defer close(release)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	client := &http.Client{Transport: newTransport(100 * time.Millisecond), CheckRedirect: CheckRedirect}
	resp, err := client.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("a server that never answers: err = nil, want a timeout")
	}
	if took := time.Since(start); took > 3*time.Second {
		t.Errorf("gave up after %v, want about the header timeout", took)
	}
}

func TestClientShipsTheThirtySecondHeaderTimeout(t *testing.T) {
	tr, ok := Client().Transport.(*http.Transport)
	if !ok || tr != Transport {
		t.Fatalf("Client().Transport = %T, want Transport", Client().Transport)
	}
	if tr.ResponseHeaderTimeout != 30*time.Second {
		t.Errorf("ResponseHeaderTimeout = %v, want 30s", tr.ResponseHeaderTimeout)
	}
}

func TestClientFollowsHTTPSRedirectsAlone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/file", http.StatusFound)
	}))
	defer srv.Close()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := Client().Do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "not HTTPS") {
		t.Errorf("a redirect to plain HTTP: err = %v, want a refusal", err)
	}
}

func TestCheckRedirectStopsAfterTen(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	if err := CheckRedirect(req, make([]*http.Request, 9)); err != nil {
		t.Errorf("after 9 redirects: err = %v, want none", err)
	}
	if err := CheckRedirect(req, make([]*http.Request, 10)); err == nil {
		t.Error("after 10 redirects: err = nil, want a stop")
	}
}
