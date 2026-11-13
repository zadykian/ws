package cld

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// releases serves /latest as a redirect to the tag.
func releases(t *testing.T, tag string) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest" {
			http.Redirect(w, r, "/tag/"+tag, http.StatusFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/latest"
}

func TestLatestReleaseIsTheTagTheRedirectEndsAt(t *testing.T) {
	if got, err := latestRelease(context.Background(), nil, releases(t, "v0.14.2")); err != nil || got != "0.14.2" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestLatestReleaseRefusesAPageThatIsNoTag(t *testing.T) {
	if _, err := latestRelease(context.Background(), nil, releases(t, "nightly")); err == nil {
		t.Error("a page that is no tag names a release")
	}
}
