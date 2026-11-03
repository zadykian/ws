package network

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestRender(t *testing.T) {
	got, err := Render("enp5s0", "10.77.0.1/30")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{`    "enp5s0":`, "      dhcp4: false", "      dhcp6: false", `      addresses: ["10.77.0.1/30"]`, "      optional: true"} {
		if !strings.Contains(string(got), line+"\n") {
			t.Errorf("the file lacks the line %q:\n%s", line, got)
		}
	}
	var parsed struct {
		Network struct {
			Ethernets map[string]struct {
				Addresses []string `yaml:"addresses"`
			} `yaml:"ethernets"`
		} `yaml:"network"`
	}
	if err := yaml.Unmarshal(got, &parsed); err != nil {
		t.Fatal(err)
	}
	if a := parsed.Network.Ethernets["enp5s0"].Addresses; len(a) != 1 || a[0] != "10.77.0.1/30" {
		t.Errorf("addresses = %q", a)
	}
}

func TestRenderQuotes(t *testing.T) {
	got, err := Render("a: b", "x\"y")
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Network struct {
			Ethernets map[string]any `yaml:"ethernets"`
		} `yaml:"network"`
	}
	if err := yaml.Unmarshal(got, &parsed); err != nil {
		t.Fatal(err)
	}
	if _, ok := parsed.Network.Ethernets["a: b"]; !ok {
		t.Errorf("the interface name was not kept whole:\n%s", got)
	}
}
