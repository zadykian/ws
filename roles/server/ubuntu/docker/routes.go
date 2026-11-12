// Package docker sets up Docker Engine from Docker's apt repository: its daemon.json,
// docker.slice, the build cache's collection, and containers' DNS through the host.
package docker

import (
	"encoding/json"
	"fmt"
)

// Route is one entry of `ip -j -4 route show table all`.
type Route struct {
	Dst   string `json:"dst"`
	Type  string `json:"type"`
	Table string `json:"table"`
	Dev   string `json:"dev"`
	Scope string `json:"scope"`
}

// ParseRoutes reads the JSON that ip prints.
func ParseRoutes(data []byte) ([]Route, error) {
	var routes []Route
	if err := json.Unmarshal(data, &routes); err != nil {
		return nil, fmt.Errorf("routes: %w", err)
	}
	return routes, nil
}
