package signoz

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIDecodesTheAnswer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"status":"success"}`)
	}))
	defer srv.Close()
	var got struct{ Status string }
	if err := (API{URL: srv.URL}).get(t.Context(), "/x", &got); err != nil || got.Status != "success" {
		t.Errorf("got %+v, %v; want success", got, err)
	}
}

func TestAPIFailsOnAnotherStatusAndSaysWhatItGot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "locked", http.StatusForbidden)
	}))
	defer srv.Close()
	err := API{URL: srv.URL}.get(t.Context(), "/api/v2/x", nil)
	if err == nil || !strings.Contains(err.Error(), "GET /api/v2/x: status 403, want 200: locked") {
		t.Errorf("err = %v, want the status and the body", err)
	}
}
