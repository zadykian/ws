package update

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// redirecting serves /latest as a redirect to location, or with status where location is empty.
func redirecting(t *testing.T, location string, status int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if location != "" {
			w.Header().Set("Location", location)
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestLatest(t *testing.T) {
	for name, tc := range map[string]struct {
		location string
		status   int
		want     string
		fails    string
	}{
		"a release":    {"https://github.com/zadykian/ws/releases/tag/v0.10.0", http.StatusFound, "v0.10.0", ""},
		"no release":   {"https://github.com/zadykian/ws/releases", http.StatusFound, "", `leads to "releases"`},
		"not a tag":    {"https://github.com/zadykian/ws/releases/tag/v01.2.3", http.StatusFound, "", "not to a release"},
		"no redirect":  {"", http.StatusOK, "", "200 OK"},
		"server error": {"", http.StatusBadGateway, "", "502 Bad Gateway"},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := latest(t.Context(), redirecting(t, tc.location, tc.status))
			if got != tc.want || (err == nil) != (tc.fails == "") || (err != nil && !strings.Contains(err.Error(), tc.fails)) {
				t.Errorf("latest = %q, %v; want %q, an error of %q", got, err, tc.want, tc.fails)
			}
		})
	}
}

func TestLatestStaysAtTheRedirect(t *testing.T) {
	// The tag's page is never fetched: a server that fails on it would fail latest otherwise.
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Redirect(w, r, "/tag/v1.2.3", http.StatusFound)
	}))
	defer srv.Close()
	if got, err := latest(t.Context(), srv.URL); got != "v1.2.3" || err != nil || calls != 1 {
		t.Errorf("latest = %q, %v after %d calls; want v1.2.3 after one", got, err, calls)
	}
}

func TestGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/file":
			_, _ = w.Write([]byte("body"))
		case "/plain":
			http.Redirect(w, r, "/file", http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	if got, err := get(t.Context(), srv.URL+"/file"); string(got) != "body" || err != nil {
		t.Errorf("get /file = %q, %v; want its body", got, err)
	}
	if _, err := get(t.Context(), srv.URL+"/plain"); err == nil || !strings.Contains(err.Error(), "not HTTPS") {
		t.Errorf("get of a redirect to HTTP = %v, want a refusal", err)
	}
	if _, err := get(t.Context(), srv.URL+"/none"); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("get of a missing file = %v, want its status", err)
	}
}

func TestChecksum(t *testing.T) {
	const sums = "aaa  ws-linux-amd64\nbbb *ws-darwin-arm64\nnot a line of sha256sum\n\n"
	for name, want := range map[string]string{"ws-linux-amd64": "aaa", "ws-darwin-arm64": "bbb", "ws-linux-arm64": ""} {
		if got, ok := checksum(sums, name); got != want || ok != (want != "") {
			t.Errorf("checksum of %s = %q, %v; want %q", name, got, ok, want)
		}
	}
}
