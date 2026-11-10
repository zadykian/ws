package wizard

import (
	"context"
	"testing"
)

func TestHostKeyScan(t *testing.T) {
	h, m := newHost(nil)
	w := Way{Host: "10.1.1.2", Port: "22"}
	if got := scanKey(context.Background(), h.M, w); got != hostLine {
		t.Errorf("scan = %q; want %q", got, hostLine)
	}
	if got, err := h.fingerprint(context.Background(), hostLine); err != nil || got != hostFP {
		t.Errorf("fingerprint = %q, %v; want %q", got, err, hostFP)
	}
	if got := string(m.Files["/l/host.pub"]); got != hostLine+"\n" {
		t.Errorf("host.pub = %q", got)
	}
}
