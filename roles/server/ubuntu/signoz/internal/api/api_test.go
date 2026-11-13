package api

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
	if err := (Client{URL: srv.URL}).Get(t.Context(), "/x", &got); err != nil || got.Status != "success" {
		t.Errorf("got %+v, %v; want success", got, err)
	}
}

func TestAPIFailsOnAnotherStatusAndSaysWhatItGot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "locked", http.StatusForbidden)
	}))
	defer srv.Close()
	err := Client{URL: srv.URL}.Get(t.Context(), "/api/v2/x", nil)
	if err == nil || !strings.Contains(err.Error(), "GET /api/v2/x: status 403, want 200: locked") {
		t.Errorf("err = %v, want the status and the body", err)
	}
}

func TestAPISendsTheBodyAsJSON(t *testing.T) {
	var got, kind string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got, kind = r.Method+" "+string(b), r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	if err := (Client{URL: srv.URL}).Call(t.Context(), http.MethodPost, "/x", []byte(`{"a":1}`), http.StatusCreated, nil); err != nil {
		t.Fatal(err)
	}
	if got != `POST {"a":1}` || kind != "application/json" {
		t.Errorf("server got %q as %q", got, kind)
	}
}
