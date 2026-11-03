package wireguard

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/wgkey"
)

// changing is a resource with one change to make.
type changing struct{}

type noop struct{}

func (noop) String() string              { return "write a file" }
func (noop) Apply(context.Context) error { return nil }

func (changing) Plan(context.Context) ([]host.Change, error) { return []host.Change{noop{}}, nil }

// idle is a resource with nothing to do.
type idle struct{}

func (idle) Plan(context.Context) ([]host.Change, error) { return nil, nil }

// machine has a key and, where wg0 is set, a wg0 that holds the key shown.
func machine(wg0Key string) (*effecttest.Machine, string) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	private, _ := wgkey.Generate(bytes.NewReader(bytes.Repeat([]byte{3}, 32)))
	public, _ := wgkey.Public(private)
	m.Files[keyFile] = []byte(private + "\n")
	if wg0Key != "" {
		m.Dirs["/sys/class/net/wg0"] = 0o755
		m.Results["wg show wg0 public-key"] = effecttest.Result{Out: []byte(wg0Key + "\n")}
	}
	return m, public
}

func descriptions(t *testing.T, w wg0) []string {
	cs, err := w.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range cs {
		got = append(got, c.String())
	}
	return got
}

func TestWg0IsLeftAloneWhereItHoldsTheKeyAndNothingChanges(t *testing.T) {
	m, public := machine("")
	m.Dirs["/sys/class/net/wg0"] = 0o755
	m.Results["wg show wg0 public-key"] = effecttest.Result{Out: []byte(public + "\n")}
	if got := descriptions(t, wg0{m: m, watch: []host.Resource{idle{}}}); len(got) != 0 {
		t.Errorf("plan = %q, want nothing", got)
	}
}

func TestWg0ReloadsForAChangedFile(t *testing.T) {
	m, public := machine("")
	m.Dirs["/sys/class/net/wg0"] = 0o755
	m.Results["wg show wg0 public-key"] = effecttest.Result{Out: []byte(public)}
	got := descriptions(t, wg0{m: m, watch: []host.Resource{idle{}, changing{}}})
	if len(got) != 1 || !strings.HasPrefix(got[0], "reload networkd for wg0") {
		t.Errorf("plan = %q, want a reload", got)
	}
}

func TestWg0IsDeletedBeforeTheReloadForAnotherKey(t *testing.T) {
	m, _ := machine("OTHER")
	got := descriptions(t, wg0{m: m})
	if len(got) != 2 || !strings.HasPrefix(got[0], "delete wg0") || !strings.HasPrefix(got[1], "reload networkd") {
		t.Errorf("plan = %q, want a deletion and a reload", got)
	}
}

func TestWg0IsDeletedForAKeyToBeMade(t *testing.T) {
	m, _ := machine("OTHER")
	delete(m.Files, keyFile)
	got := descriptions(t, wg0{m: m})
	if len(got) != 2 || !strings.HasPrefix(got[0], "delete wg0") {
		t.Errorf("plan = %q, want a deletion and a reload", got)
	}
}

func TestWg0MissingIsReloadedAndNeverDeleted(t *testing.T) {
	m, _ := machine("")
	got := descriptions(t, wg0{m: m})
	if len(got) != 1 || !strings.Contains(got[0], "wg0 is missing") {
		t.Errorf("plan = %q, want a reload", got)
	}
}
