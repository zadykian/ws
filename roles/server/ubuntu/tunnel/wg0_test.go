package tunnel

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/roles/internal/wgkey"
)

// machine has a key and, where wg0Key is set, a wg0 that holds the key shown.
func machine(wg0Key string) (*effecttest.Machine, string) {
	m := effecttest.NewMachine()
	private, _ := wgkey.Generate(bytes.NewReader(bytes.Repeat([]byte{3}, 32)))
	public, _ := wgkey.Public(private)
	m.Files[keyFile] = []byte(private + "\n")
	if wg0Key != "" {
		m.Dirs["/sys/class/net/wg0"] = 0o755
		m.Results["wg show wg0 public-key"] = effecttest.Result{Out: []byte(wg0Key + "\n")}
	}
	return m, public
}

// newWG0 is WG0 on m with three tries, which counts its pauses in slept.
func newWG0(m *effecttest.Machine, out *effecttest.Out, slept *int) WG0 {
	return WG0{M: m, Out: out, Tries: 3, Sleep: func(time.Duration) { *slept++ }}
}

func headlines(t *testing.T, r resource.Resource) []string {
	cs, err := r.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range cs {
		got = append(got, c.String())
	}
	return got
}

func TestWG0CheckDueWhereMissing(t *testing.T) {
	m, _ := machine("")
	got := headlines(t, newWG0(m, &effecttest.Out{}, new(int)))
	if len(got) != 2 || !strings.Contains(got[0], "wg0 is missing") || !strings.HasPrefix(got[1], "wait for wg0") {
		t.Errorf("plan = %q, want the reload and the wait", got)
	}
}

func TestWG0DeletedOnAnotherKey(t *testing.T) {
	m, public := machine("OTHER")
	m.Results["ip link delete wg0"] = effecttest.Result{}
	m.Results["networkctl reload"] = effecttest.Result{}
	out := &effecttest.Out{}
	cs, err := newWG0(m, out, new(int)).Plan(t.Context())
	if err != nil || len(cs) != 3 {
		t.Fatalf("plan = %v, %v; want the deletion, the reload and the wait", cs, err)
	}
	for _, c := range cs[:2] {
		if err := c.Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	m.Results["wg show wg0 public-key"] = effecttest.Result{Out: []byte(public + "\n")}
	if err := cs[2].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, "root ip link delete wg0") || !strings.Contains(strings.Join(out.Lines, "\n"), public) {
		t.Errorf("calls %q, said %q; want wg0 deleted and the key said", m.Calls, out.Lines)
	}
}

func TestWG0WaitsForTheKey(t *testing.T) {
	m, _ := machine("OTHER")
	slept := 0
	cs, err := newWG0(m, &effecttest.Out{}, &slept).Plan(t.Context())
	if err != nil || len(cs) == 0 {
		t.Fatalf("plan = %v, %v", cs, err)
	}
	if err := cs[len(cs)-1].Apply(t.Context()); err == nil || slept != 2 {
		t.Errorf("wait = %v after %d pauses, want a failure after 2", err, slept)
	}
}

func TestWG0HoldingTheKeyPlansNothing(t *testing.T) {
	m, public := machine(".")
	m.Results["wg show wg0 public-key"] = effecttest.Result{Out: []byte(public + "\n")}
	if got := headlines(t, newWG0(m, &effecttest.Out{}, new(int))); len(got) != 0 {
		t.Errorf("plan = %q, want nothing", got)
	}
}

func TestWG0WithoutAKeyFails(t *testing.T) {
	m, _ := machine("")
	delete(m.Files, keyFile)
	if _, err := newWG0(m, &effecttest.Out{}, new(int)).Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "tunnel key makes it") {
		t.Errorf("plan = %v, want the key missing", err)
	}
}

func TestWG0ReportsWhereWgShowFails(t *testing.T) {
	m, _ := machine("OTHER")
	m.Results["wg show wg0 public-key"] = effecttest.Result{Err: errors.New("no device")}
	if _, err := newWG0(m, &effecttest.Out{}, new(int)).Plan(t.Context()); err == nil {
		t.Error("Plan = nil, want the error")
	}
}

func TestWG0WaitsThroughAMissingWg0(t *testing.T) {
	m, public := machine("")
	m.Results["networkctl reload"] = effecttest.Result{}
	slept := 0
	cs, err := newWG0(m, &effecttest.Out{}, &slept).Plan(t.Context())
	if err != nil || len(cs) != 2 {
		t.Fatalf("Plan = %v, %v", cs, err)
	}
	m.Results["wg show wg0 public-key"] = effecttest.Result{Out: []byte(public)}
	if err := cs[1].Apply(t.Context()); err != nil || slept != 0 {
		t.Errorf("wait = %v after %d pauses; want done at once", err, slept)
	}
}
