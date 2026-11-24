package tunnel

import (
	"bytes"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/wgkey"
)

func newKey(m *effecttest.Machine) Key {
	m.Files["/etc/group"] = []byte("systemd-network:x:998:\n")
	return Key{M: m, Out: &effecttest.Out{}, Random: bytes.NewReader(bytes.Repeat([]byte{7}, 32))}
}

func TestKeyIsMadeWhereMissing(t *testing.T) {
	m := effecttest.NewMachine()
	k := newKey(m)
	cs, err := k.Plan(t.Context())
	if err != nil || len(cs) != 1 {
		t.Fatalf("Plan = %v, %v; want one change", cs, err)
	}
	if strings.Contains(cs[0].String(), "AAAA") || !strings.Contains(cs[0].String(), "content not shown") {
		t.Errorf("change = %q", cs[0])
	}
	if err := cs[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	private := strings.TrimSpace(string(m.Files[keyFile]))
	if _, err := wgkey.Public(private); err != nil {
		t.Errorf("the key written is no private key: %v", err)
	}
	if got := m.Owners[keyFile]; got != [2]int{0, 998} {
		t.Errorf("owner = %v, want root:998", got)
	}
	public, _ := wgkey.Public(private)
	said := strings.Join(k.Out.(*effecttest.Out).Lines, "\n")
	if !strings.Contains(said, "values.yml: "+public) || strings.Contains(said, private) {
		t.Errorf("said %q, want the public key alone", said)
	}
}

func TestKeyKeepsItsContentAndSetsItsMode(t *testing.T) {
	m := effecttest.NewMachine()
	private, _ := wgkey.Generate(bytes.NewReader(bytes.Repeat([]byte{9}, 32)))
	m.Files[keyFile] = []byte(private + "\n")
	m.Root[keyFile] = effect.Attrs{Mode: 0o600}
	cs, err := newKey(m).Plan(t.Context())
	if err != nil || len(cs) != 1 {
		t.Fatalf("Plan = %v, %v; want one change", cs, err)
	}
	if err := cs[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	got, _ := m.RootAttrs(keyFile)
	if string(m.Files[keyFile]) != private+"\n" || got != (effect.Attrs{GID: 998, Mode: 0o640}) {
		t.Errorf("file = %q, attrs %+v", m.Files[keyFile], got)
	}
}

func TestKeyRightIsLeftAlone(t *testing.T) {
	m := effecttest.NewMachine()
	private, _ := wgkey.Generate(bytes.NewReader(bytes.Repeat([]byte{9}, 32)))
	m.Files[keyFile] = []byte(private + "\n")
	m.Root[keyFile] = effect.Attrs{GID: 998, Mode: 0o640}
	if cs, err := newKey(m).Plan(t.Context()); err != nil || len(cs) != 0 {
		t.Errorf("Plan = %v, %v; want nothing", cs, err)
	}
}

func TestKeyRefusesAFileWithNoKey(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[keyFile] = []byte("not a key\n")
	if _, err := newKey(m).Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "holds no WireGuard private key") {
		t.Errorf("Plan = %v, want a refusal", err)
	}
}

func TestKeyRefusesAMachineWithoutTheGroup(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/group"] = []byte("root:x:0:\n")
	if _, err := (Key{M: m}).Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "systemd-network is missing") {
		t.Errorf("Plan = %v, want the group missing", err)
	}
}
