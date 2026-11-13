package fetch_test

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host/fetch"
)

func TestDownload(t *testing.T) {
	m := effecttest.NewMachine()
	m.Pages["https://x/y.deb"] = []byte("deb")
	d := fetch.Download{M: m, URL: "https://x/y.deb", Path: "/tmp/y.deb", Mode: 0o644,
		SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("deb")))}
	if got := applyAll(t, d); len(got) != 1 || got[0] != "download https://x/y.deb to /tmp/y.deb" {
		t.Errorf("first plan = %q, want a download", got)
	}
	if got := applyAll(t, d); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
	m.Files[d.Path] = []byte("other")
	d.Mode = 0o600
	if got := applyAll(t, d); len(got) != 1 || !strings.HasPrefix(got[0], "download") {
		t.Errorf("a changed file: plan %q, want a download", got)
	}
}

func TestDownloadWrongSumKeepsTheFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Pages["u"] = []byte("evil")
	m.Files["/f"] = []byte("good")
	changes, _ := fetch.Download{M: m, URL: "u", Path: "/f", SHA256: "00"}.Plan(t.Context())
	if err := changes[0].Apply(t.Context()); err == nil || string(m.Files["/f"]) != "good" {
		t.Errorf("Apply = %v, file %q; want a failure and the old file", err, m.Files["/f"])
	}
}

func TestDownloadSetsAttributesOfAMatchingFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/f"] = []byte("deb")
	d := fetch.Download{M: m, Path: "/f", Mode: 0o600, SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("deb")))}
	if got := applyAll(t, d); len(got) != 1 || !strings.HasPrefix(got[0], "set owner 0:0 and mode 600") {
		t.Errorf("plan = %q, want an owner and mode change", got)
	}
}
