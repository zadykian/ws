package claude

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"
)

// cldReleases is the page that GitHub redirects to the newest release's tag.
const cldReleases = "https://github.com/zadykian/cld/releases/latest"

// latestRelease returns the version that url leads to, such as 0.14.2 for .../releases/tag/v0.14.2,
// as cld update finds it. A nil client is an http.Client with a timeout.
func latestRelease(ctx context.Context, client *http.Client, url string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	var err error
	var req *http.Request
	for attempt := range 3 {
		if req, err = http.NewRequestWithContext(ctx, http.MethodHead, url, nil); err != nil {
			return "", err
		}
		var resp *http.Response
		if resp, err = client.Do(req); err == nil {
			_ = resp.Body.Close() // only the status matters
			if resp.StatusCode == http.StatusOK {
				tag := path.Base(resp.Request.URL.Path)
				if !strings.HasPrefix(tag, "v") {
					return "", fmt.Errorf("%s leads to %s, not a release's tag", url, resp.Request.URL)
				}
				return tag[1:], nil
			}
			err = fmt.Errorf("%s: %s", url, resp.Status)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}
	return "", err
}
