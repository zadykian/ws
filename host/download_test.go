package host_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestDownload(t *testing.T) {
	m := effecttest.NewMachine()
	m.Pages["https://x/y.deb"] = []byte("deb")
	d := host.Download{M: m, URL: "https://x/y.deb", Path: "/tmp/y.deb", Mode: 0o644,
		SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("deb")))}
	if got := applyAll(t, d); len(got) != 1 || got[0] != "download https://x/y.deb to /tmp/y.deb" {
		t.Errorf("first plan = %q, want a download", got)
	}
	if got := applyAll(t, d); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}
