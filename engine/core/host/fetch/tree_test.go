package fetch_test

import (
	"archive/tar"
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host/fetch"
)

const (
	treeURL  = "https://x/1.2/t-amd64.tar"
	treeFile = fetch.CacheDir + "/t-amd64.tar"
	treeDir  = "/opt/t"
)

func treeArchive(t *testing.T) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, name := range []string{"t-1.2/", "t-1.2/f"} {
		hdr := &tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: 1}
		if strings.HasSuffix(name, "/") {
			hdr.Typeflag, hdr.Size, hdr.Mode = tar.TypeDir, 0, 0o755
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if hdr.Size > 0 {
			_, _ = tw.Write([]byte("x"))
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func treeFixture(t *testing.T) (*effecttest.Machine, fetch.Tree) {
	m := effecttest.NewMachine()
	body := treeArchive(t)
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Pages[treeURL] = body
	return m, fetch.Tree{M: m, Path: treeDir, Strip: 1, Source: fetch.PinnedFile{M: m, Version: "1.2",
		SHA256: map[string]string{"amd64": sumOf(string(body))},
		URL:    func(version, arch string) string { return "https://x/" + version + "/t-" + arch + ".tar" }}}
}

func TestTreeInstallsStagedAndMovedWhole(t *testing.T) {
	m, tree := treeFixture(t)
	m.Files[treeDir+"/old"] = []byte("old")
	m.Dirs[treeDir] = 0o755
	got := applyAll(t, tree)
	want := []string{"create directory " + fetch.CacheDir + ", owner 0:0, mode 755",
		"download " + treeURL + " to " + treeFile, "install " + treeURL + " in " + treeDir}
	if !slices.Equal(got, want) {
		t.Fatalf("plan = %q, want %q", got, want)
	}
	if string(m.Files[treeDir+"/f"]) != "x" || string(m.Files[treeDir+"/.ws-source"]) != treeURL+"\n" {
		t.Errorf("Files = %v", m.Files)
	}
	for _, left := range []string{treeDir + "/old", treeFile, "/opt/.t.staging", "/opt/.t.staging/f"} {
		if _, ok := m.Files[left]; ok {
			t.Errorf("%s was left", left)
		}
	}
	if _, ok := m.Dirs["/opt/.t.staging"]; ok {
		t.Error("the staging directory was left")
	}
	if got := applyAll(t, tree); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestTreeWithTheMarkerAsksNothing(t *testing.T) {
	m, tree := treeFixture(t)
	m.Files[treeDir+"/.ws-source"] = []byte(treeURL + "\n")
	tree.Source.SHA256 = nil
	tree.Source.Lookup = func(context.Context, string) (string, error) {
		t.Error("Lookup asked for a current tree")
		return "", nil
	}
	if got := applyAll(t, tree); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "download ") }) {
		t.Errorf("calls = %q, want no fetch", m.Calls)
	}
}

func TestTreeWithAStaleMarkerInstallsAgain(t *testing.T) {
	m, tree := treeFixture(t)
	m.Files[treeDir+"/.ws-source"] = []byte("https://x/1.1/t-amd64.tar\n")
	if got := applyAll(t, tree); len(got) != 3 || !strings.HasPrefix(got[2], "install ") {
		t.Errorf("plan = %q, want the install", got)
	}
	if string(m.Files[treeDir+"/.ws-source"]) != treeURL+"\n" {
		t.Error("the marker is stale")
	}
}

func TestTreeRemovesWhatACutRunLeft(t *testing.T) {
	m, tree := treeFixture(t)
	m.Files[treeDir+"/.ws-source"] = []byte(treeURL + "\n")
	m.Files[treeFile] = []byte("part")
	m.Dirs["/opt/.t.staging"] = 0o755
	m.Files["/opt/.t.staging/f"] = []byte("x")
	want := []string{"remove " + treeFile, "remove /opt/.t.staging"}
	if got := applyAll(t, tree); !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q", got, want)
	}
	if got := applyAll(t, tree); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestTreeUnpacksAfreshOverAStagingAFailedRunLeft(t *testing.T) {
	m, tree := treeFixture(t)
	m.Dirs["/opt/.t.staging"] = 0o755
	m.Files["/opt/.t.staging/stray"] = []byte("x")
	applyAll(t, tree)
	if _, ok := m.Files[treeDir+"/stray"]; ok {
		t.Error("the stray file of the old staging reached the tree")
	}
}
