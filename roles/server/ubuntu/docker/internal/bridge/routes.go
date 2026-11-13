// Package bridge checks Docker's default bridge and the routes around it, and makes the ufw rules
// that let containers reach the resolver.
package bridge

import (
	"encoding/json"
	"fmt"
)

// route is one entry of `ip -j -4 route show table all`.
type route struct {
	Dst   string `json:"dst"`
	Type  string `json:"type"`
	Table string `json:"table"`
	Dev   string `json:"dev"`
	Scope string `json:"scope"`
}

// parseRoutes reads the JSON that ip prints.
func parseRoutes(data []byte) ([]route, error) {
	var routes []route
	if err := json.Unmarshal(data, &routes); err != nil {
		return nil, fmt.Errorf("routes: %w", err)
	}
	return routes, nil
}
