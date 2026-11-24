package cli

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/ubuntu/values"
)

// The files that the units of docker write.
const (
	dockerDaemon   = "/etc/docker/daemon.json"
	dockerSlice    = "/etc/systemd/system/docker.slice"
	dockerResolved = "/etc/systemd/resolved.conf.d/docker.conf"
)

// dockerMachine is a machine with 16 GiB, 8 CPUs and 500 GiB under Docker's root.
// dockerd passes the daemon.json it checks.
func dockerMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	for _, d := range []string{"/etc", "/etc/systemd", "/etc/systemd/system", "/var/lib/docker"} {
		m.Dirs[d] = 0o755
	}
	m.Files["/proc/meminfo"] = []byte("MemTotal:       16777216 kB\nMemFree: 5 kB\n")
	m.Files["/proc/cpuinfo"] = []byte("processor\t: 0\nprocessor\t: 1\nprocessor\t: 2\nprocessor\t: 3\n" +
		"processor\t: 4\nprocessor\t: 5\nprocessor\t: 6\nprocessor\t: 7\n")
	m.Results["stat -f -c %S %b /var/lib/docker"] = effecttest.Result{Out: []byte("4096 131072000\n")}
	m.Results["dockerd --validate --config-file "+dockerDaemon+".ws-check"] = effecttest.Result{}
	return m
}

// dockerValues returns the values with yaml's keys set over the defaults.
func dockerValues(t *testing.T, yaml string) values.Values {
	v, err := values.Parse([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// dockerWrites runs docker's units that write files on a new dockerMachine with v, and returns
// the machine.
func dockerWrites(t *testing.T, v values.Values) *effecttest.Machine {
	m := dockerMachine()
	units, err := kind.Load(kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v, Self: selfPath}, os.DirFS(rolesDir))
	if err != nil {
		t.Fatal(err)
	}
	writes := map[string]bool{"file/docker-resolved": true, "file/docker-slice": true, "service/docker": true}
	for _, u := range units {
		if !writes[u.Name] {
			continue
		}
		p, err := u.Step.Plan(t.Context())
		if err != nil {
			t.Fatalf("plan %s: %v", u.Name, err)
		}
		if err := u.Step.Apply(t.Context(), p); err != nil {
			t.Fatalf("apply %s: %v", u.Name, err)
		}
	}
	return m
}

// sameJSON fails the test unless the file path of m is valid JSON and the golden file's bytes,
// which main's encoder wrote for the same values.
func sameJSON(t *testing.T, m *effecttest.Machine, path, golden string) {
	want, err := os.ReadFile("testdata/" + golden) //nolint:gosec // a golden file of the tree, named by the test
	if err != nil {
		t.Fatal(err)
	}
	got := m.Files[path]
	if !json.Valid(got) {
		t.Errorf("%s is not JSON:\n%s", path, got)
	}
	if string(got) != string(want) {
		t.Errorf("%s =\n%s\nwant testdata/%s:\n%s", path, got, golden, want)
	}
}

// The defaults: an age of 90 days in hours, and the cache's limits as 10%, 80% and 20% of the
// disk in whole GiB plus one.
func TestDockerDaemonJSONHasTheDefaults(t *testing.T) {
	m := dockerWrites(t, dockerValues(t, ""))
	sameJSON(t, m, dockerDaemon, "docker-daemon.json")
}

// An age of 0 days leaves the age's rule out, and set sizes stand in for the disk's share.
func TestDockerDaemonJSONTakesTheCacheValues(t *testing.T) {
	m := dockerWrites(t, dockerValues(t, "WS_DOCKER_BUILD_CACHE_KEEP_DAYS: 0\n"+
		"WS_DOCKER_BUILD_CACHE_RESERVED_SPACE: 2GB\nWS_DOCKER_BUILD_CACHE_MAX_USED_SPACE: 9GB\n"+
		"WS_DOCKER_BUILD_CACHE_MIN_FREE_SPACE: 3GB\n"))
	sameJSON(t, m, dockerDaemon, "docker-daemon-set.json")
}

func TestDockerSliceIsHalfTheMachineOrTheValues(t *testing.T) {
	const head = "# Written by github.com/zadykian/ws, whose next run replaces any change made here.\n" +
		"[Unit]\nDescription=Docker's daemon and its containers\n\n[Slice]\n"
	for _, c := range []struct{ yaml, limits string }{
		{"", "MemoryMax=8192M\nCPUQuota=400%\nMemorySwapMax=0\n"},
		{"WS_DOCKER_MEMORY_MAX: 1G\nWS_DOCKER_CPU_QUOTA: 50%\nWS_DOCKER_MEMORY_SWAP_MAX: 2G\n",
			"MemoryMax=1G\nCPUQuota=50%\nMemorySwapMax=2G\n"},
	} {
		m := dockerWrites(t, dockerValues(t, c.yaml))
		if got := string(m.Files[dockerSlice]); got != head+c.limits {
			t.Errorf("values %q: %s =\n%s\nwant\n%s", c.yaml, dockerSlice, got, head+c.limits)
		}
	}
}

func TestDockerResolvedListensOnTheBridge(t *testing.T) {
	m := dockerWrites(t, dockerValues(t, "WS_DOCKER_BRIDGE_IP: 172.18.0.1\n"))
	const want = "[Resolve]\nDNSStubListenerExtra=172.18.0.1\n"
	if got := string(m.Files[dockerResolved]); len(got) < len(want) || got[len(got)-len(want):] != want {
		t.Errorf("%s =\n%s\nwant it to end\n%s", dockerResolved, got, want)
	}
	var j struct {
		DNS []string `json:"dns"`
		Bip string   `json:"bip"`
	}
	if err := json.Unmarshal(m.Files[dockerDaemon], &j); err != nil || len(j.DNS) != 1 || j.DNS[0] != "172.18.0.1" ||
		j.Bip != "172.18.0.1/16" {
		t.Errorf("daemon.json's dns %q and bip %q (%v), want the bridge's address", j.DNS, j.Bip, err)
	}
}

// The policy of FORWARD, as the unit reads it and sets it, as root.
const (
	// asRoot is how a run state's command runs as root, with no session's env.
	asRoot = "env HOME=/root env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS " +
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "
	forwardIsAccept  = asRoot + "sh -c iptables -w -S FORWARD | grep -qx -- '-P FORWARD ACCEPT'"
	forwardToAccept  = asRoot + "iptables -w -P FORWARD ACCEPT"
	forwardUnitName  = "state/docker-forward-policy"
	dockerForwardKey = "ip-forward-no-drop"
)

// A first run of Docker on a machine with forwarding off sets the policy to drop, and the option
// keeps a later start from doing it again.
func TestDockerDaemonJSONKeepsTheForwardPolicy(t *testing.T) {
	m := dockerWrites(t, dockerValues(t, ""))
	var j map[string]any
	if err := json.Unmarshal(m.Files[dockerDaemon], &j); err != nil || j[dockerForwardKey] != true {
		t.Errorf("%s: %s is %v (%v), want true", dockerDaemon, dockerForwardKey, j[dockerForwardKey], err)
	}
}

// Where Docker's first start set the policy to drop, the unit sets it back to accept, and leaves a
// policy that is accept alone.
func TestDockerForwardPolicyUndoesDrop(t *testing.T) {
	m := dockerMachine()
	m.Results[forwardIsAccept] = effecttest.Result{Err: errors.New("exit status 1")}
	m.Results[forwardToAccept] = effecttest.Result{}
	units, err := kind.Load(kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: dockerValues(t, ""), Self: selfPath}, os.DirFS(rolesDir))
	if err != nil {
		t.Fatal(err)
	}
	got := applyUnit(t, units, forwardUnitName)
	if len(got) != 1 || !strings.Contains(got[0], "FORWARD ACCEPT") || !slices.Contains(m.Calls, "run "+forwardToAccept) {
		t.Errorf("plan %q, calls %q; want the policy set to accept", got, m.Calls)
	}
	m.Results[forwardIsAccept] = effecttest.Result{}
	if got := applyUnit(t, units, forwardUnitName); got != nil {
		t.Errorf("a policy that is accept plans %q, want none", got)
	}
}

// The policy is set after Docker's first start, which is what sets it to drop.
func TestDockerForwardPolicyComesAfterTheService(t *testing.T) {
	units, err := kind.Load(kind.Env{M: dockerMachine(), U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: dockerValues(t, ""), Self: selfPath}, os.DirFS(rolesDir))
	if err != nil {
		t.Fatal(err)
	}
	names := kind.Names(units)
	if at, svc := slices.Index(names, forwardUnitName), slices.Index(names, "service/docker"); at < 0 || at < svc {
		t.Errorf("%s runs at %d, want it after service/docker at %d", forwardUnitName, at, svc)
	}
}
