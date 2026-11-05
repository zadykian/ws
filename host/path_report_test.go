package host_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/step"
)

func TestReportsOfPaths(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/f"] = []byte("old\n")
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
	}
	for _, tt := range tests {
		if got := reportOf(t, tt.r); got != tt.want {
			t.Errorf("item = %+v, want %+v", got, tt.want)
		}
	}
}
