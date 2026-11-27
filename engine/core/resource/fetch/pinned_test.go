package fetch_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/fetch"
	"github.com/zadykian/ws/engine/core/resource/resourcetest"
)

var sumOf = resourcetest.SumOf

func pinned(m *effecttest.Machine, sums map[string]string) fetch.PinnedFile {
	return fetch.PinnedFile{M: m, Version: "1.2", Path: "/usr/local/bin/x", Mode: 0o755, SHA256: sums,
		URL: func(version, arch string) string { return "https://x/" + version + "/x-" + arch }}
}

func TestPinnedFilePicksTheArchitecturesURLAndSum(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("arm64\n")}
	m.Pages["https://x/1.2/x-arm64"] = []byte("arm")
	p := pinned(m, map[string]string{"amd64": sumOf("amd"), "arm64": sumOf("arm")})
	if got := applyAll(t, p); len(got) != 1 || got[0] != "download https://x/1.2/x-arm64 to /usr/local/bin/x" {
		t.Errorf("first plan = %q, want the arm64 download", got)
	}
	if got := applyAll(t, p); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestPinnedFileReadsTheArchitectureByItsArch(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["uname -m"] = effecttest.Result{Out: []byte("arm64\n")}
	m.Pages["https://x/1.2/x-arm64"] = []byte("arm")
	p := pinned(m, map[string]string{"arm64": sumOf("arm")})
	p.Arch = fetch.Uname
	if got := applyAll(t, p); len(got) != 1 || got[0] != "download https://x/1.2/x-arm64 to /usr/local/bin/x" {
		t.Errorf("plan = %q, want the arm64 download, with no dpkg", got)
	}
}

func TestPinnedFileAllReadsNoArchitecture(t *testing.T) {
	m := effecttest.NewMachine()
	m.Pages["https://x/1.2/x-all"] = []byte("any")
	p := pinned(m, map[string]string{"all": sumOf("any")})
	if got := applyAll(t, p); len(got) != 1 || !strings.Contains(got[0], "x-all") {
		t.Errorf("plan = %q, want the all download", got)
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "dpkg") {
			t.Errorf("a call read dpkg: %q", m.Calls)
		}
	}
}

func TestPinnedFileWithNoPinFailsBeforeAURL(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("riscv64\n")}
	p := pinned(m, map[string]string{"amd64": sumOf("amd")})
	p.URL = func(string, string) string { t.Error("URL called without a pin"); return "" }
	_, err := p.Plan(t.Context())
	if err == nil || !strings.Contains(err.Error(), "no SHA-256 pinned for the architecture riscv64") {
		t.Errorf("Plan err = %v, want the missing pin named", err)
	}
}
