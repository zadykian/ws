package docker_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/server/ubuntu/docker"
)

func TestRootlessDropInsPinTheSliceAndTheControllers(t *testing.T) {
	override, err := docker.RootlessOverride()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Slice=docker.slice", "ROOTLESSKIT_NET=slirp4netns", "PORT_DRIVER=builtin"} {
		if !strings.Contains(string(override), want) {
			t.Errorf("the override lacks %q", want)
		}
	}
	delegate, err := docker.Delegate()
	if err != nil || !strings.Contains(string(delegate), "Delegate=cpu io memory pids") {
		t.Errorf("Delegate = %q, %v; want the io controller", delegate, err)
	}
}
