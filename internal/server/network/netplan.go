package network

import (
	"fmt"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// settings is the part of `netplan get` that the check reads.
type settings struct {
	Network struct {
		Renderer  string         `yaml:"renderer"`
		Ethernets map[string]any `yaml:"ethernets"`
	} `yaml:"network"`
}

// NetplanProblems returns what keeps netplan from setting up the direct link on port alone.
// generated is the result of netplan's configure, run on root, and got that of `netplan get`
// there. The settings may hold secrets, such as Wi-Fi passwords, so no message quotes them. An
// entry of another name that fits the port is refused: it keeps a networkd file of its own, which
// networkd may take first. A match: on the port's own entry that may fit other interfaces is
// refused too, as they would take the direct link's settings.
func NetplanProblems(generated, got Result, root string, port Port) []string {
	if generated.RC != 0 {
		return []string{fmt.Sprintf("netplan rejects its files with the direct link's: %s.", errorLine(generated, root))}
	}
	if got.RC != 0 || strings.Contains(got.Stderr, "Command failed") {
		return []string{fmt.Sprintf("netplan can't read its settings: %s.", errorLine(got, root))}
	}
	var s settings
	if err := yaml.Unmarshal([]byte(got.Stdout), &s); err != nil {
		return []string{"netplan get printed settings that aren't YAML."}
	}
	name := port.Name
	var problems []string
	entry, _ := s.Network.Ethernets[name].(map[string]any)
	renderer := "networkd"
	if r, ok := entry["renderer"]; ok {
		renderer = text(r)
	} else if s.Network.Renderer != "" {
		renderer = s.Network.Renderer
	}
	if renderer != "networkd" {
		problems = append(problems, fmt.Sprintf("netplan hands %s to %s, not networkd, which the role sets up the direct link with.", name, renderer))
	}
	match, _ := entry["match"].(map[string]any)
	_, hasMatch := entry["match"]
	switch {
	case !fits(match, port):
		problems = append(problems, fmt.Sprintf("netplan's entry %s has a match: that %s doesn't fit, so the direct link would go to another interface.", name, name))
	case hasMatch && !alone(match):
		problems = append(problems, fmt.Sprintf("netplan's entry %s has a match: that may fit other interfaces too, whose DHCP the direct link would turn off: match %s's MAC address, or a name without a glob.", name, name))
	}
	for _, other := range sortedKeys(s.Network.Ethernets) {
		if other == name {
			continue
		}
		definition, ok := s.Network.Ethernets[other].(map[string]any)
		if !ok {
			continue
		}
		// An entry without a match: matches its own name, which may be the port's alternative one.
		theirs := map[string]any{"name": other}
		if _, has := definition["match"]; has {
			theirs, _ = definition["match"].(map[string]any)
		}
		if fits(theirs, port) {
			problems = append(problems, fmt.Sprintf("netplan's entry %s fits %s, and keeps a networkd file apart from the direct link's: move its settings to %s, or narrow its match: to another interface.", other, name, name))
		}
	}
	return problems
}

// sortedKeys returns the keys of m in order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
