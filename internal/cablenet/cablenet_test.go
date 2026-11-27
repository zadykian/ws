package cablenet_test

import (
	"testing"

	"github.com/zadykian/ws/internal/cablenet"
)

func TestTheCablesEnds(t *testing.T) {
	for name, c := range map[string][2]string{
		"Net":          {cablenet.Net.String(), "10.77.0.0/30"},
		"Server":       {cablenet.Server().String(), "10.77.0.1"},
		"ServerPrefix": {cablenet.ServerPrefix().String(), "10.77.0.1/30"},
		"Mac":          {cablenet.Mac().String(), "10.77.0.2"},
		"Mask":         {cablenet.Mask(), "255.255.255.252"},
	} {
		if c[0] != c[1] {
			t.Errorf("%s = %s, want %s", name, c[0], c[1])
		}
	}
}

func TestBothEndsLieInTheSubnet(t *testing.T) {
	if !cablenet.Net.Contains(cablenet.Server()) || !cablenet.Net.Contains(cablenet.Mac()) {
		t.Error("an end lies outside the subnet")
	}
}
