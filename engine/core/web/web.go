// Package web holds the rules of ws's checked downloads: a header timeout, and redirects to HTTPS
// alone. Neither caps the whole transfer, so that a large file on a slow line finishes.
package web

import (
	"fmt"
	"net/http"
	"time"
)

// HeaderTimeout is how long a server may stay silent before a request fails.
const HeaderTimeout = 30 * time.Second

// maxRedirects is the most redirects a request follows.
const maxRedirects = 10

// Transport is Go's default, which takes a proxy from the environment, but gives up on a server
// that has not begun to answer within HeaderTimeout.
var Transport = newTransport(HeaderTimeout)

func newTransport(header time.Duration) *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone() //nolint:forcetypeassert // Go's default is a *http.Transport
	t.ResponseHeaderTimeout = header
	return t
}

// CheckRedirect refuses a redirect that is not HTTPS, and stops after ten.
func CheckRedirect(req *http.Request, via []*http.Request) error {
	if req.URL.Scheme != "https" {
		return fmt.Errorf("redirected to %s, which is not HTTPS", req.URL)
	}
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	return nil
}

// Client is a client with both rules.
func Client() *http.Client {
	return &http.Client{Transport: Transport, CheckRedirect: CheckRedirect}
}
