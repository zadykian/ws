package kind

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/ufw"
)

// loadState loads one state unit, state/a, of the fields given.
func loadState(t *testing.T, e Env, fields string) []Unit {
	return loadOne(t, e, "[state.a]\n"+fields, nil)
}

// stateRefused fails the test unless the state of the fields fails to load with an error that
// holds want.
func stateRefused(t *testing.T, e Env, fields, want string) {
	_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[state.a]\n" + fields}))
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("load of\n%s= %v, want %q", fields, err, want)
	}
}

func TestEveryStateFieldIsAState(t *testing.T) {
	kind := reflect.TypeFor[state]()
	for f := range reflect.TypeFor[stateSpec]().Fields() {
		if f.Anonymous {
			continue
		}
		if !f.Type.Implements(kind) {
			t.Errorf("stateSpec.%s is %s, want a state", f.Name, f.Type)
		}
		if tag := f.Tag.Get("toml"); tag == "" || tag != strings.ToLower(tag) {
			t.Errorf("stateSpec.%s has the toml tag %q, want the state's key", f.Name, tag)
		}
	}
}

func TestStateTakesExactlyOne(t *testing.T) {
	e, _, _, _ := testEnv(false)
	const keys = "exactly one of linger, subids, purged, http, ufw, routes, addr, left, run, reboot, locale, timezone, swap"
	stateRefused(t, e, "after = \"\"\n", keys+", and this one sets 0")
	stateRefused(t, e, "linger = \"u\"\npurged = \"p\"\n", keys+", and this one sets 2")
	stateRefused(t, e, "ufw = {}\nlinger = \"u\"\n", "this one sets 2")
	stateRefused(t, e, "linger = \"-u\"\n", `linger: linger "-u" is no user's name`)
	stateRefused(t, e, "purged = \"-y\"\n", `purged: "-y" is no package's name`)
}

func TestStateLinger(t *testing.T) {
	e, m, u, _ := testEnv(true)
	withAccount(m)
	units := loadState(t, e, "linger = \"u\"\n")
	if got, want := headlines(t, units[0].Step), []string{"enable lingering for u", "start user@1001.service"}; !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q", got, want)
	}
	m.Files["/var/lib/systemd/linger/u"] = nil
	u.States = map[string]effect.UnitState{"user@1001.service": {Active: true}}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan with u lingering and its manager up = %q, want none", got)
	}
}

func TestStateLingerSkipsUnbooted(t *testing.T) {
	e, m, _, out := testEnv(false)
	if got := headlines(t, loadState(t, e, "linger = \"u\"\n")[0].Step); got != nil ||
		slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "id ") }) ||
		!slices.Equal(out.Lines, []string{"say: lingering for u: skipped, systemd is not running"}) {
		t.Errorf("plan = %q, calls %q, lines %q; want none, no lookup, and the skip said once", got, m.Calls, out.Lines)
	}
}

func TestStateSubIDs(t *testing.T) {
	for _, c := range []struct {
		subuid, subgid string
		want           string
	}{
		{"u:100000:65536\n", "v:1:2\nu:200000:65536\n", ""},
		{"u:100000:65536\n", "u:100000:1000\n", "root usermod --add-subgids 100000-165535 u"},
		{"", "", "root usermod --add-subuids 100000-165535 --add-subgids 100000-165535 u"},
	} {
		e, m, _, _ := testEnv(false)
		m.Files["/etc/subuid"], m.Files["/etc/subgid"] = []byte(c.subuid), []byte(c.subgid)
		m.Results["usermod --add-subgids 100000-165535 u"] = effecttest.Result{}
		m.Results["usermod --add-subuids 100000-165535 --add-subgids 100000-165535 u"] = effecttest.Result{}
		units := loadState(t, e, "subids = \"u\"\n")
		if err := applyOnce(t, units[0].Step); err != nil {
			t.Fatal(err)
		}
		got := slices.DeleteFunc(slices.Clone(m.Calls), func(call string) bool { return !strings.HasPrefix(call, "root usermod") })
		if want := slices.DeleteFunc([]string{c.want}, func(s string) bool { return s == "" }); !slices.Equal(got, want) {
			t.Errorf("subuid %q, subgid %q: calls = %q, want %q", c.subuid, c.subgid, got, want)
		}
	}
}

func TestStatePurged(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Packages = effecttest.Packages{"p": {Version: "1", Status: "ii"}}
	units := loadState(t, e, "purged = \"p\"\n")
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"purge p"}) {
		t.Errorf("plan = %q, want p purged", got)
	}
	m.Packages = effecttest.Packages{}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan without p = %q, want none", got)
	}
}

// answering serves 503 for the first busy asks, then 200.
func answering(t *testing.T, busy int32) string {
	var asked atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if asked.Add(1) <= busy {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	t.Cleanup(s.Close)
	return s.URL + "/"
}

func TestStateHTTPWaits(t *testing.T) {
	e, _, _, _ := testEnv(true)
	if got := headlines(t, loadState(t, e, "http = \""+answering(t, 0)+"\"\n")[0].Step); got != nil {
		t.Errorf("plan of a URL that answers = %q, want none", got)
	}
	url := answering(t, 2)
	if got := headlines(t, loadState(t, e, "http = \""+url+"\"\n")[0].Step); !slices.Equal(got, []string{"wait for an answer at " + url}) {
		t.Errorf("plan of a busy URL = %q, want the wait", got)
	}
	h := health{m: e.M, out: e.Out, url: url, tries: 3}
	if err := (await{h}).Apply(t.Context()); err != nil {
		t.Errorf("wait for the third answer = %v, want none", err)
	}
	h.url, h.tries = answering(t, 5), 2
	if err := (await{h}).Apply(t.Context()); err == nil || !strings.Contains(err.Error(), "after 2 tries") {
		t.Errorf("wait past the tries = %v, want the tries named", err)
	}
	stateRefused(t, e, "http = \"127.0.0.1:1\"\n", "no http or https URL")
}

func TestStateHTTPSkipsUnbooted(t *testing.T) {
	e, _, _, out := testEnv(false)
	if got := headlines(t, loadState(t, e, "http = \"http://127.0.0.1:1/\"\n")[0].Step); got != nil ||
		!slices.Equal(out.Lines, []string{"say: http://127.0.0.1:1/: skipped, systemd is not running"}) {
		t.Errorf("plan = %q, lines %q; want none, and the skip said", got, out.Lines)
	}
}

func TestStateUFWOwnsItsComment(t *testing.T) {
	e, m, _, _ := testEnv(true)
	// The rule from 10.0.0.0/8 carries the comment DNS, 444e53 in hex, and the one from any too.
	m.Files[ufw.UserRules] = []byte("### tuple ### allow any 53 172.17.0.1 any 0.0.0.0/0 in_docker0 comment=444e53\n" +
		"### tuple ### allow any 53 172.17.0.1 any 10.0.0.0/8 in_docker0 comment=444e53\n" +
		"### tuple ### allow tcp 22 0.0.0.0/0 any 0.0.0.0/0 in comment=535348\n")
	units := loadState(t, e, "[state.a.ufw]\ncomment = \"DNS\"\nrules = [\n"+
		"  { action = \"allow\", proto = \"any\", to = \"172.17.0.1\", to_port = \"53\", dir = \"in\", interface = \"docker0\" },\n"+
		"  { action = \"allow\", proto = \"any\", to = \"172.17.0.1\", to_port = \"53\", dir = \"in\", interface = \"br-+\" },\n]\n")
	want := []string{
		"ufw allow in on br-+ from any to 172.17.0.1 port 53",
		"ufw delete allow in on docker0 from 10.0.0.0/8 to 172.17.0.1 port 53",
	}
	if got := headlines(t, units[0].Step); !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q", got, want)
	}
	stateRefused(t, e, "ufw = { rules = [] }\n", "takes a comment")
	stateRefused(t, e, "ufw = { comment = \"c\", rules = [{ action = \"allow\", proto = \"tcp\", dir = \"up\" }] }\n",
		"ufw: rule 1: ufw rule")
}

func TestStateRoutesAndAddrRefuse(t *testing.T) {
	e, m, _, _ := testEnv(true)
	m.Results["ip -j -4 route show table all"] = effecttest.Result{Out: []byte(`[{"dst":"10.1.5.0/24","dev":"wg0","table":"7"}]`)}
	units := loadState(t, e, "routes = { pool = \"10.1.0.0/16\", net = \"172.17.0.1/16\", dev = \"docker0\", hint = \"Move it.\" }\n")
	_, err := units[0].Step.Plan(t.Context())
	if err == nil || !strings.Contains(err.Error(), "10.1.5.0/24 dev wg0 table 7") || !strings.HasSuffix(err.Error(), "Move it.") {
		t.Errorf("plan of the routes = %v, want the route refused with the hint", err)
	}
	// No interface takes this name, so it holds no address.
	units = loadState(t, e, "addr = { iface = \"ws-none0\", want = \"10.9.0.1/24\", hint = \"Reboot.\" }\n")
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"ws-none0 holds no address, not 10.9.0.1/24"}) {
		t.Errorf("plan of the address = %q, want the missing address", got)
	}
	stateRefused(t, e, "routes = { pool = \"10.1.0.0\", net = \"172.17.0.1/16\", dev = \"docker0\" }\n", "routes: pool:")
	stateRefused(t, e, "routes = { pool = \"10.1.0.0/16\", net = \"172.17.0.1/16\" }\n", "takes dev")
	stateRefused(t, e, "addr = { iface = \"docker0\", want = \"x\" }\n", "addr: want:")
}

func TestStateLeft(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Files["/etc/group"] = []byte("lxd:x:100:v,u\n")
	units := loadState(t, e, "left = { user = \"u\", group = \"lxd\" }\n")
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"remove u from the lxd group"}) {
		t.Errorf("plan = %q, want u out of lxd", got)
	}
	stateRefused(t, e, "left = { user = \"u\" }\n", `group "" is no user's name`)
}

// asU is how a run state runs a command as u: through runuser, in u's session, with the system's
// PATH.
const asU = "runuser -u u -- env -C /home/u HOME=/home/u env XDG_RUNTIME_DIR=/run/user/1001 " +
	"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1001/bus " + systemPath + " "

func TestStateRunCreates(t *testing.T) {
	e, m, _, _ := testEnv(true)
	withAccount(m)
	m.Results[asU+"sh -c x"] = effecttest.Result{}
	units := loadState(t, e, "run = { user = \"u\", command = [\"sh\", \"-c\", \"x\"], creates = [\"/home/u/.x\", \"/home/u/.y\"] }\n")
	m.Files["/home/u/.x"] = nil
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"run sh -c x as u"}) {
		t.Errorf("plan with /home/u/.y missing = %q, want the run", got)
	}
	if err := applyOnce(t, units[0].Step); err != nil || !slices.Contains(m.Calls, "root "+asU+"sh -c x") {
		t.Errorf("apply = %v, calls %q; want the command run as u", err, m.Calls)
	}
	m.Files["/home/u/.y"] = nil
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan with both paths = %q, want none", got)
	}
}

func TestStateRunCheck(t *testing.T) {
	e, m, _, _ := testEnv(true)
	withAccount(m)
	m.Results[asU+"ws x check"] = effecttest.Result{}
	units := loadState(t, e, "run = { user = \"u\", command = [\"ws\", \"x\"], check = [\"ws\", \"x\", \"check\"] }\n")
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan where the check passes = %q, want none", got)
	}
	m.Results[asU+"ws x check"] = effecttest.Result{Err: errors.New("exit status 1")}
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"run ws x as u"}) {
		t.Errorf("plan where the check fails = %q, want the run", got)
	}
}

func TestStateRunAsRoot(t *testing.T) {
	e, m, _, _ := testEnv(true)
	const asRoot = "env HOME=/root env XDG_RUNTIME_DIR=/run/user/0 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/0/bus " +
		systemPath + " sh -c x"
	m.Results[asRoot] = effecttest.Result{}
	units := loadState(t, e, "run = { user = \"root\", command = [\"sh\", \"-c\", \"x\"], creates = [\"/x\"] }\n")
	if err := applyOnce(t, units[0].Step); err != nil || !slices.Contains(m.Calls, "run "+asRoot) {
		t.Errorf("apply = %v, calls %q; want the command run as root, in /root", err, m.Calls)
	}
}

func TestStateRunPutsItsPathFirst(t *testing.T) {
	e, m, _, _ := testEnv(true)
	withAccount(m)
	const first = "runuser -u u -- env -C /home/u HOME=/home/u env XDG_RUNTIME_DIR=/run/user/1001 " +
		"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1001/bus " +
		"PATH=/home/u/.local/bin:/home/u/.t:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "
	m.Results[first+"ws x check"] = effecttest.Result{Err: errors.New("exit status 1")}
	m.Results[first+"ws x"] = effecttest.Result{}
	units := loadState(t, e, "run = { user = \"u\", path = [\"/home/u/.local/bin\", \"/home/u/.t\"], "+
		"command = [\"ws\", \"x\"], check = [\"ws\", \"x\", \"check\"] }\n")
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"run ws x as u"}) {
		t.Errorf("plan where the check fails = %q, want the run", got)
	}
	if err := applyOnce(t, units[0].Step); err != nil || !slices.Contains(m.Calls, "root "+first+"ws x") {
		t.Errorf("apply = %v, calls %q; want the command run with its path first", err, m.Calls)
	}
}

func TestStateRunSkipsUnbooted(t *testing.T) {
	e, m, _, out := testEnv(false)
	units := loadState(t, e, "run = { user = \"u\", command = [\"sh\"], creates = [\"/x\"] }\n")
	if got := headlines(t, units[0].Step); got != nil || slices.Contains(m.Calls, "stat /x") ||
		!slices.Equal(out.Lines, []string{"say: run sh as u: skipped, systemd is not running"}) {
		t.Errorf("plan = %q, calls %q, lines %q; want nothing read, and the skip said", got, m.Calls, out.Lines)
	}
}

func TestStateRunTakesCreatesOrCheck(t *testing.T) {
	e, _, _, _ := testEnv(false)
	stateRefused(t, e, "run = { user = \"u\", command = [\"sh\"] }\n", "exactly one of creates and check")
	stateRefused(t, e, "run = { user = \"u\", command = [\"sh\"], creates = [\"/x\"], check = [\"true\"] }\n",
		"exactly one of creates and check")
	stateRefused(t, e, "run = { user = \"u\", creates = [\"/x\"] }\n", "takes a command")
	stateRefused(t, e, "run = { user = \"u\", command = [\"sh\"], creates = [\"x\"] }\n", "not a clean absolute path")
	stateRefused(t, e, "run = { command = [\"sh\"], creates = [\"/x\"] }\n", "is no user's name")
}

func TestStateRunRefusesABadPath(t *testing.T) {
	e, _, _, _ := testEnv(false)
	for _, p := range []string{"bin", "/a/../b", "/a:/b", "/a/"} {
		stateRefused(t, e, "run = { user = \"u\", path = [\""+p+"\"], command = [\"sh\"], creates = [\"/x\"] }\n",
			"free of ':'")
	}
}
