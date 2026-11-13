// Package web reads the URLs that the role pins from a vendor.
package web

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
)

// Read reads a URL. A vendor's server can fail for a moment, so curl tries again, five times at
// 10 seconds.
func Read(ctx context.Context, m effect.Machine, url string) ([]byte, error) {
	body, err := m.Run(ctx, "curl", "-fsSL", "--retry", "5", "--retry-delay", "10", "--retry-all-errors", url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	return body, nil
}
