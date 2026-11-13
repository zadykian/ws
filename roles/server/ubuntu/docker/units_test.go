package docker_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/server/ubuntu/docker"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func TestUnitFilesCarryTheValues(t *testing.T) {
	d := values.Default().Docker
	d.MemoryMax, d.CPUQuota = "8000M", "200%"
	slice, err := docker.Slice(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"MemoryMax=8000M\n", "CPUQuota=200%\n", "MemorySwapMax=0\n"} {
		if !strings.Contains(string(slice), want) {
			t.Errorf("docker.slice = %q, want %q", slice, want)
		}
	}
	resolved, err := docker.Resolved(d)
	if err != nil || !strings.HasSuffix(string(resolved), "DNSStubListenerExtra=172.17.0.1\n") {
		t.Errorf("resolved drop-in = %q, %v; want the bridge's address", resolved, err)
	}
	override, err := docker.Override()
	if err != nil || !strings.HasSuffix(string(override), "[Service]\nSlice=docker.slice\n") {
		t.Errorf("override = %q, %v; want the slice", override, err)
	}
}
