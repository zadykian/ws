package fetch_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/fetch"
)

func sumOf(body string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(body))) }

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
	if _, err := p.Report(t.Context()); err == nil {
		t.Error("Report succeeded without a pin")
	}
}

func TestPinnedFileReportsAFileOfOtherContent(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Files["/usr/local/bin/x"] = []byte("other")
	items, err := pinned(m, map[string]string{"amd64": sumOf("amd")}).Report(t.Context())
	if err != nil || len(items) != 1 || items[0].Conflict == "" {
		t.Errorf("Report = %v, %v; want one conflict", items, err)
	}
}

func TestPinnedFileAsksLookupForTheSumWhereNoneIsPinned(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Pages["https://x/1.2/x-amd64"] = []byte("amd")
	var asked []string
	p := pinned(m, nil)
	p.Lookup = func(_ context.Context, url string) (string, error) {
		asked = append(asked, url)
		return sumOf("amd"), nil
	}
	if got := applyAll(t, p); len(got) != 1 || !strings.Contains(got[0], "x-amd64") {
		t.Errorf("plan = %q, want the amd64 download", got)
	}
	if len(asked) == 0 || asked[0] != "https://x/1.2/x-amd64" {
		t.Errorf("Lookup asked for %q", asked)
	}
	p.SHA256 = map[string]string{"amd64": sumOf("amd")}
	asked = nil
	if _, err := p.Plan(t.Context()); err != nil || len(asked) != 0 {
		t.Errorf("with a pin: err %v, Lookup asked for %q", err, asked)
	}
}

func TestPinnedFileFailsWhereLookupFails(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	p := pinned(m, nil)
	p.Lookup = func(context.Context, string) (string, error) { return "", errors.New("no list") }
	if _, err := p.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "no list") {
		t.Errorf("Plan err = %v, want Lookup's", err)
	}
}
