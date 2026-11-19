package cli

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/roles/server/values"
)

// The account of docker's tests, and the files that the units of docker write for root and for it.
const (
	dockerHome     = "/home/zadykian"
	dockerDaemon   = "/etc/docker/daemon.json"
	dockerSlice    = "/etc/systemd/system/docker.slice"
	dockerResolved = "/etc/systemd/resolved.conf.d/docker.conf"
	dockerRootless = dockerHome + "/.config/docker/daemon.json"
)

// dockerMachine is an unbooted machine, as CI's container is, with 16 GiB, 8 CPUs, 500 GiB under
// Docker's root and the account zadykian. dockerd passes each daemon.json it checks.
func dockerMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	for _, d := range []string{"/etc", "/etc/systemd", "/etc/systemd/system", "/var/lib/docker"} {
		m.Dirs[d] = 0o755
	}
	m.Files["/proc/meminfo"] = []byte("MemTotal:       16777216 kB\nMemFree: 5 kB\n")
	m.Files["/proc/cpuinfo"] = []byte("processor\t: 0\nprocessor\t: 1\nprocessor\t: 2\nprocessor\t: 3\n" +
		"processor\t: 4\nprocessor\t: 5\nprocessor\t: 6\nprocessor\t: 7\n")
	m.Results["stat -f -c %S %b /var/lib/docker"] = effecttest.Result{Out: []byte("4096 131072000\n")}
	m.Results["id -u zadykian"] = effecttest.Result{Out: []byte("1001\n")}
	m.Results["id -g zadykian"] = effecttest.Result{Out: []byte("1001\n")}
	m.Results["getent passwd zadykian"] = effecttest.Result{Out: []byte("zadykian:x:1001:1001::" + dockerHome + ":/bin/bash\n")}
	m.Dirs[dockerHome] = 0o750
	m.Owners[dockerHome] = [2]int{1001, 1001}
	for _, p := range []string{dockerDaemon, dockerRootless} {
		m.Results["dockerd --validate --config-file "+p+".ws-check"] = effecttest.Result{}
	}
	return m
}

// dockerValues returns the values of an account's machine, with yaml's keys set over the defaults.
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
	writes := map[string]bool{"file/docker-resolved": true, "file/docker-slice": true, "service/docker": true,
		"service/docker-rootless": true}
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

// A run stops at its first failure, so the account leaves the docker group, which amounts to root,
// before any unit of the system's daemon may fail. Its own daemon waits on the system's.
func TestDockerAccountLeavesTheGroupFirst(t *testing.T) {
	e := kind.Env{M: dockerMachine(), U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: dockerValues(t, ""), Self: selfPath}
	units, err := kind.Load(e, os.DirFS(rolesDir))
	if err != nil {
		t.Fatal(err)
	}
	names := kind.Names(units)
	left, routes := slices.Index(names, "state/docker-left"), slices.Index(names, "state/docker-routes")
	if left < 0 || routes < left {
		t.Errorf("state/docker-left runs at %d, state/docker-routes at %d; want the group left first", left, routes)
	}
	if left >= 0 && units[left].Step.WaitsOn() != "" {
		t.Errorf("state/docker-left waits on %q, want none", units[left].Step.WaitsOn())
	}
	if i := slices.Index(names, "state/docker-subids"); i < 0 || units[i].Step.WaitsOn() != "state/docker-bridge" {
		t.Errorf("state/docker-subids at %d does not wait on state/docker-bridge", i)
	}
}

// The account's unit is Docker's own, and the account's. Its PATH names no caller's directory, such
// as mise's shims, which the setup tool wrote in.
func TestDockerAccountUnitIsDockersWithTheSystemsPath(t *testing.T) {
	const unit = dockerHome + "/.config/systemd/user/docker.service"
	m := dockerWrites(t, dockerValues(t, ""))
	got := string(m.Files[unit])
	for _, line := range []string{"ExecStart=/usr/bin/dockerd-rootless.sh\n", "Requires=dbus.socket\n",
		"Environment=PATH=/usr/bin:/sbin:/usr/sbin:/usr/local/sbin:/usr/local/bin:/bin\n", "WantedBy=default.target\n"} {
		if !strings.Contains(got, line) {
			t.Errorf("%s lacks %q:\n%s", unit, line, got)
		}
	}
	if m.Owners[unit] != [2]int{1001, 1001} {
		t.Errorf("%s is owned by %v, want the account's 1001", unit, m.Owners[unit])
	}
}

// The account's daemon is started by the run that writes its unit, then enabled, and only then is
// its context made the account's current one.
func TestDockerAccountDaemonGoesBeforeItsLinkAndContext(t *testing.T) {
	e := kind.Env{M: dockerMachine(), U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: dockerValues(t, ""), Self: selfPath}
	units, err := kind.Load(e, os.DirFS(rolesDir))
	if err != nil {
		t.Fatal(err)
	}
	names := kind.Names(units)
	order := []string{"service/docker-rootless", "state/docker-rootless-enable", "state/docker-rootless-context"}
	for i, name := range order[1:] {
		at := slices.Index(names, name)
		if at < 0 || at < slices.Index(names, order[i]) || units[at].Step.WaitsOn() != order[i] {
			t.Errorf("%s at %d does not wait on %s", name, at, order[i])
		}
	}
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
	sameJSON(t, m, dockerRootless, "docker-rootless.json")
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
		for _, path := range []string{dockerSlice, dockerHome + "/.config/systemd/user/docker.slice"} {
			if got := string(m.Files[path]); got != head+c.limits {
				t.Errorf("values %q: %s =\n%s\nwant\n%s", c.yaml, path, got, head+c.limits)
			}
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
