package rootless

import (
	"strings"
	"testing"
)

func TestRootlessDropInsPinTheSliceAndTheControllers(t *testing.T) {
	override, err := overrideDropIn()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Slice=docker.slice", "ROOTLESSKIT_NET=slirp4netns", "PORT_DRIVER=builtin"} {
		if !strings.Contains(string(override), want) {
			t.Errorf("the override lacks %q", want)
		}
	}
	delegate, err := delegateDropIn()
	if err != nil || !strings.Contains(string(delegate), "Delegate=cpu io memory pids") {
		t.Errorf("Delegate = %q, %v; want the io controller", delegate, err)
	}
}
