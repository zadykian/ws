package object_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/ubuntu/roles/incus/object"
)

// aclKind is incus's network ACL, as its docs give the commands (howto/network_acls).
var aclKind = object.Kind{
	Noun:   "network ACL",
	List:   []string{"network", "acl", "list"},
	Create: []string{"network", "acl", "create", object.NameArg},
	Edit:   []string{"network", "acl", "edit", object.NameArg},
	Keys:   []string{"description", "ingress", "egress", "config"},
}

const (
	listLine = "incus network acl list --format json"
	// oneACL is what incus lists for an ACL it holds: the declared keys, the keys that incus
	// fills, and a second object that the name must not match.
	oneACL = `[{"name":"other","description":"","egress":[{"action":"drop","state":"enabled"}]},
{"name":"ws-egress","description":"the egress","used_by":["/1.0/networks/incusbr0"],"project":"default",
"ingress":null,"egress":[{"action":"drop","state":"enabled","destination":"10.0.0.0/8","description":"set by incus"}],
"config":{"user.note":"a"}}]`
)

func declared() map[string]any {
	return map[string]any{
		"description": "the egress",
		"egress":      []any{map[string]any{"action": "drop", "state": "enabled", "destination": "10.0.0.0/8"}},
	}
}

func acl(m *effecttest.Machine, want map[string]any) object.Object {
	return object.Object{M: m, Kind: aclKind, Name: "ws-egress", Want: want}
}

func plan(t *testing.T, r resource.Resource) []resource.Change {
	t.Helper()
	changes, err := r.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return changes
}

func TestAMissingObjectIsCreatedFromItsKeys(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[listLine] = effecttest.Result{Out: []byte("[]")}
	m.Results["incus network acl create ws-egress"] = effecttest.Result{}
	o := acl(m, declared())
	o.First = map[string]any{"type": "bridge"}
	changes := plan(t, o)
	if len(changes) != 1 {
		t.Fatalf("changes = %v, want one create", changes)
	}
	if got := changes[0].String(); !strings.HasPrefix(got, "create the network ACL ws-egress\n") {
		t.Errorf("change = %q", got)
	}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	want := "description: the egress\negress:\n    - action: drop\n      destination: 10.0.0.0/8\n      state: enabled\ntype: bridge\n"
	if len(m.Inputs) != 1 || m.Inputs[0] != want {
		t.Errorf("stdin = %q, want %q", m.Inputs, want)
	}
}

func TestAnObjectWithItsKeysIsLeftAlone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[listLine] = effecttest.Result{Out: []byte(oneACL)}
	o := acl(m, declared())
	o.First = map[string]any{"type": "bridge"}
	if changes := plan(t, o); len(changes) != 0 {
		t.Errorf("changes = %v, want none: incus's own keys and the create's alone differ from nothing", changes)
	}
}

func TestAnEmptyListMatchesAKeyThatIncusLeavesOut(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[listLine] = effecttest.Result{Out: []byte(`[{"name":"ws-egress","description":"x"}]`)}
	want := map[string]any{"description": "x", "ingress": []any{}, "config": map[string]any{}}
	if changes := plan(t, acl(m, want)); len(changes) != 0 {
		t.Errorf("changes = %v, want none", changes)
	}
}

func TestADifferingKeyIsPatchedIntoWhatIncusHolds(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[listLine] = effecttest.Result{Out: []byte(oneACL)}
	m.Results["incus network acl edit ws-egress"] = effecttest.Result{}
	want := declared()
	want["egress"] = []any{map[string]any{"action": "drop", "state": "enabled", "destination": "10.0.0.0/8,172.16.0.0/12"}}
	changes := plan(t, acl(m, want))
	if len(changes) != 1 {
		t.Fatalf("changes = %v, want one edit", changes)
	}
	if head, _, _ := strings.Cut(changes[0].String(), "\n"); head != "edit the network ACL ws-egress: egress" {
		t.Errorf("first line = %q", head)
	}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	// The body keeps the key that Incus filled in config, and drops used_by and project.
	body := m.Inputs[0]
	for _, has := range []string{"user.note: a", "destination: 10.0.0.0/8,172.16.0.0/12", "description: the egress"} {
		if !strings.Contains(body, has) {
			t.Errorf("the edit's YAML lacks %q: %s", has, body)
		}
	}
	for _, lacks := range []string{"used_by", "project", "set by incus", "type:"} {
		if strings.Contains(body, lacks) {
			t.Errorf("the edit's YAML holds %q: %s", lacks, body)
		}
	}
}

func TestASecondRunAfterAnEditChangesNothing(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[listLine] = effecttest.Result{Out: []byte(`[{"name":"ws-egress","description":"old"}]`)}
	m.After = map[string]map[string]effecttest.Result{
		"incus network acl edit ws-egress": {listLine: {Out: []byte(`[{"name":"ws-egress","description":"the egress"}]`)}},
	}
	m.Results["incus network acl edit ws-egress"] = effecttest.Result{}
	o := acl(m, map[string]any{"description": "the egress"})
	changes := plan(t, o)
	if len(changes) != 1 {
		t.Fatalf("changes = %v, want one", changes)
	}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	if again := plan(t, o); len(again) != 0 {
		t.Errorf("a second plan holds %v, want none", again)
	}
}

func TestANumberIsComparedAsText(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[listLine] = effecttest.Result{Out: []byte(`[{"name":"ws-egress","config":{"size":1000000}}]`)}
	want := map[string]any{"config": map[string]any{"size": "1000000"}}
	if changes := plan(t, acl(m, want)); len(changes) != 0 {
		t.Errorf("changes = %v, want none", changes)
	}
}

func TestACheckRefusesAnObjectThatExists(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[listLine] = effecttest.Result{Out: []byte(oneACL)}
	o := acl(m, declared())
	o.Check = func(map[string]any) error { return errors.New("it cannot move") }
	_, err := o.Plan(t.Context())
	if err == nil || !strings.Contains(err.Error(), "the network ACL ws-egress: it cannot move") {
		t.Errorf("err = %v", err)
	}
}

func TestAFailedListIsNoMissingObject(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[listLine] = effecttest.Result{Err: errors.New("incus: the daemon is down")}
	if _, err := acl(m, declared()).Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "daemon is down") {
		t.Errorf("err = %v", err)
	}
}

func TestAListThatIsNoArrayFails(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[listLine] = effecttest.Result{Out: []byte("Name  DESCRIPTION\n")}
	if _, err := acl(m, declared()).Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "no JSON array") {
		t.Errorf("err = %v", err)
	}
}

func TestACommandWithoutTheNameIsRefused(t *testing.T) {
	m := effecttest.NewMachine()
	o := acl(m, declared())
	o.Kind.Edit = []string{"network", "acl", "edit"}
	if _, err := o.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), object.NameArg) {
		t.Errorf("err = %v", err)
	}
}

func TestACreateTakesItsArgsAfterTheName(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["incus storage list --format json"] = effecttest.Result{Out: []byte("[]")}
	m.Results["incus storage create ws zfs"] = effecttest.Result{}
	pool := object.Object{M: m, Name: "ws", Args: []string{"zfs"}, Want: map[string]any{"description": "d"},
		First: map[string]any{"config": map[string]any{"source": "/dev/vdb1"}},
		Kind: object.Kind{Noun: "storage pool", List: []string{"storage", "list"},
			Create: []string{"storage", "create", object.NameArg}, Edit: []string{"storage", "edit", object.NameArg}}}
	changes := plan(t, pool)
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	if want := "config:\n    source: /dev/vdb1\ndescription: d\n"; m.Inputs[0] != want {
		t.Errorf("stdin = %q, want %q", m.Inputs[0], want)
	}
}

func TestAMissingKeyIsPatchedIn(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[listLine] = effecttest.Result{Out: []byte(`[{"name":"ws-egress","config":{"a":"1"}}]`)}
	want := map[string]any{"config": map[string]any{"a": "1", "b.c": "2", "d": "3", "e": "4", "f": "5", "g": "6"}}
	changes := plan(t, acl(m, want))
	if len(changes) != 1 {
		t.Fatalf("changes = %v, want one", changes)
	}
	head, _, _ := strings.Cut(changes[0].String(), "\n")
	if want := "edit the network ACL ws-egress: config.b.c, config.d, config.e, config.f, …"; head != want {
		t.Errorf("first line = %q, want %q", head, want)
	}
}
