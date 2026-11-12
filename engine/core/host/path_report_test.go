package host_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
)

func TestReportsOfPaths(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/f"] = []byte("old\n")
	m.Dirs["/d"] = 0o700
	m.Links["/l"] = "/elsewhere"
	const asRoot = "owner 0:0, mode 644"
	const changed = "its owner or mode changed"
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte("old\n")))
	tests := []struct {
		r    step.Reporter
		want step.Item
	}{
		{host.Download{M: m, Path: "/new", Mode: 0o644},
			step.Item{Kind: step.FileItem, Name: "/new", Want: asRoot, Found: "missing"}},
		{host.Download{M: m, Path: "/f", SHA256: "0", Mode: 0o644},
			step.Item{Kind: step.FileItem, Name: "/f", Want: asRoot, Found: asRoot + ", other content",
				Conflict: "differs from the download ws pins: a change by hand, or another version of ws"}},
		{host.Download{M: m, Path: "/f", SHA256: sum, Mode: 0o644},
			step.Item{Kind: step.FileItem, Name: "/f", Want: asRoot, Found: asRoot}},
		{host.Download{M: m, Path: "/f", SHA256: sum, Mode: 0o600},
			step.Item{Kind: step.FileItem, Name: "/f", Want: "owner 0:0, mode 600", Found: asRoot, Conflict: changed}},
		{host.Dir{M: m, Path: "/new", Mode: 0o755},
			step.Item{Kind: step.DirItem, Name: "/new", Want: "owner 0:0, mode 755", Found: "missing"}},
		{host.Dir{M: m, Path: "/d", Mode: 0o755, CreateOnly: true},
			step.Item{Kind: step.DirItem, Name: "/d", Want: "present", Found: "owner 0:0, mode 700"}},
		{host.Dir{M: m, Path: "/d", Mode: 0o755},
			step.Item{Kind: step.DirItem, Name: "/d", Want: "owner 0:0, mode 755", Found: "owner 0:0, mode 700", Conflict: changed}},
		{host.Dir{M: m, Path: "/d", Mode: 0o700},
			step.Item{Kind: step.DirItem, Name: "/d", Want: "owner 0:0, mode 700", Found: "owner 0:0, mode 700"}},
		{host.Link{M: m, Path: "/new", Target: "/t"},
			step.Item{Kind: step.LinkItem, Name: "/new", Want: "/t", Found: "missing"}},
		{host.Link{M: m, Path: "/l", Target: "/t"},
			step.Item{Kind: step.LinkItem, Name: "/l", Want: "/t", Found: "/elsewhere", Conflict: "not the link ws makes"}},
		{host.Link{M: m, Path: "/l", Target: "/elsewhere"},
			step.Item{Kind: step.LinkItem, Name: "/l", Want: "/elsewhere", Found: "/elsewhere"}},
		{host.Link{M: m, Path: "/f", Target: "/t"},
			step.Item{Kind: step.LinkItem, Name: "/f", Want: "/t", Found: "not a link", Conflict: "not the link ws makes"}},
		{host.Absent{M: m, Path: "/f"}, step.Item{Kind: step.FileItem, Name: "/f", Want: "missing", Found: "present"}},
		{host.Absent{M: m, Path: "/new"}, step.Item{Kind: step.FileItem, Name: "/new", Want: "missing", Found: "missing"}},
	}
	for _, tt := range tests {
		if got := reportOf(t, tt.r); got != tt.want {
			t.Errorf("item = %+v, want %+v", got, tt.want)
		}
	}
}
