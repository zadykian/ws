package docker

import (
	"crypto/sha256"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/ufw"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

func fakeSteps(t *testing.T, user string) (*effecttest.Machine, []step.Step) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	m.Dirs["/var/lib/docker"] = 0o710
	m.Files["/proc/meminfo"] = []byte("MemTotal:       16384000 kB\n")
	m.Files["/proc/cpuinfo"] = []byte("processor\t: 0\nprocessor\t: 1\n")
	m.Files["/etc/os-release"] = []byte("VERSION_CODENAME=resolute\n")
	m.Files["/etc/passwd"] = []byte("ws:x:1000:1000::/home/ws:/bin/sh\n")
	m.Files["/etc/group"] = []byte("docker:x:999:\n")
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	var rules string
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Files["/var/lib/systemd/linger/ws"] = nil
	// The tool's unit and link exist, as the fake runs no tool: rootlessInstall has its own tests.
	m.Files["/home/ws/.config/systemd/user/docker.service"] = nil
	m.Files["/home/ws/.config/systemd/user/default.target.wants/docker.service"] = nil
	m.Dirs["/run/user/1000/systemd/private"] = 0o755
	m.Results["runuser -u ws -- env -C /home/ws HOME=/home/ws env XDG_RUNTIME_DIR=/run/user/1000 "+
		"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus systemctl --user show --property=ActiveState --value docker.service"] =
		effecttest.Result{Out: []byte("active\n")}
	for _, verb := range []string{"daemon-reload", "restart docker.service"} {
		m.Results["runuser -u ws -- env -C /home/ws HOME=/home/ws env XDG_RUNTIME_DIR=/run/user/1000 "+
			"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus systemctl --user "+verb] = effecttest.Result{}
	}
	m.Files["/etc/subuid"] = []byte("ws:100000:65536\n")
	m.Files["/etc/subgid"] = []byte("ws:100000:65536\n")
	v := values.Default()
	v.User, v.Home = user, "/home/"+user
	var installed strings.Builder
	names := packages(v)
	for _, n := range names {
		installed.WriteString(n + " ii\n")
	}
	m.Results["dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n "+strings.Join(names, " ")] =
		effecttest.Result{Out: []byte(installed.String())}
	m.Results["ip -j -4 route show table all"] = effecttest.Result{Out: []byte("[]")}
	m.Results["dockerd --validate --config-file /etc/docker/daemon.json.ws-check"] = effecttest.Result{}
	m.Results["dockerd --validate --config-file /home/ws/.config/docker/daemon.json.ws-check"] = effecttest.Result{}
	for _, b := range []string{"docker0", "br-+"} {
		rules += "### tuple ### allow any 53 172.17.0.1 any 0.0.0.0/0 in_" + b + " comment=444e532066726f6d20636f6e7461696e657273\n"
	}
	m.Files[ufw.UserRules] = []byte(rules)
	m.Pages[keyURL] = []byte("key")
	pin := ascSHA256
	ascSHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte("key")))
	t.Cleanup(func() { ascSHA256 = pin })
	held := func() ([]netip.Prefix, error) { return []netip.Prefix{netip.MustParsePrefix("172.17.0.1/16")}, nil }
	size := func(string) (uint64, error) { return 100 << 30, nil }
	return m, steps(v, m, &effecttest.Units{}, &effecttest.Out{}, size, held)
}

func TestStepsWaitOnTheirPackagesInOrder(t *testing.T) {
	_, ss := fakeSteps(t, "ws")
	var names, after []string
	for _, s := range ss {
		names, after = append(names, s.Name()), append(after, s.WaitsOn())
	}
	wantNames := []string{repositoryStep, packagesStep, routesStep, membersStep, configStep, bridgeStep, rootlessStep}
	wantAfter := []string{"", repositoryStep, packagesStep, packagesStep, routesStep, configStep, configStep}
	if !slices.Equal(names, wantNames) || !slices.Equal(after, wantAfter) {
		t.Errorf("steps = %q after %q, want %q after %q", names, after, wantNames, wantAfter)
	}
	_, ss = fakeSteps(t, "root")
	if len(ss) != len(wantNames)-2 {
		t.Errorf("root has %d steps, want %d: root needs no group and no daemon of its own", len(ss), len(wantNames)-2)
	}
}

func TestStepsApplyOnceAndThenPlanNothing(t *testing.T) {
	m, ss := fakeSteps(t, "ws")
	out := &effecttest.Out{}
	if changed, err := step.Run(t.Context(), ss, out, false); err != nil || !changed {
		t.Fatalf("first run = %v, %v; want changes", changed, err)
	}
	if changed, err := step.Run(t.Context(), ss, out, false); err != nil || changed {
		t.Errorf("second run = %v, %v; want nothing to change", changed, err)
	}
	for path, want := range map[string]string{
		"/etc/docker/daemon.json":                            `"cgroup-parent": "docker.slice"`,
		"/etc/systemd/resolved.conf.d/docker.conf":           "DNSStubListenerExtra=172.17.0.1",
		"/etc/systemd/system/docker.slice":                   "MemoryMax=8000M",
		"/etc/systemd/system/docker.service.d/override.conf": "docker.slice",
		"/etc/apt/sources.list.d/docker.sources":             "Suites: resolute",
		"/home/ws/.config/docker/daemon.json":                `"dns": [` + "\n" + `    "10.0.2.3"`,
		"/home/ws/.config/systemd/user/docker.slice":         "MemoryMax=8000M",
		"/etc/systemd/system/user@.service.d/delegate.conf":  "Delegate=cpu io memory pids",
	} {
		if !strings.Contains(string(m.Files[path]), want) {
			t.Errorf("%s = %q, want it to hold %q", path, m.Files[path], want)
		}
	}
}

func TestStepsTakeTheUserOutOfTheDockerGroup(t *testing.T) {
	m, ss := fakeSteps(t, "ws")
	m.Files["/etc/group"] = []byte("docker:x:999:ws\n")
	m.Results["gpasswd --delete ws docker"] = effecttest.Result{}
	// The fake leaves /etc/group unchanged, so the run's check after applying still sees the user.
	_, err := step.Run(t.Context(), ss, &effecttest.Out{}, false)
	if err == nil || !strings.Contains(err.Error(), "remove ws from the docker group") {
		t.Errorf("run = %v, want the group's step to plan the removal", err)
	}
	if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "gpasswd --delete ws docker") }) {
		t.Errorf("calls = %q, want gpasswd", m.Calls)
	}
}

func TestTheRootlessFilesRefuseALink(t *testing.T) {
	m, ss := fakeSteps(t, "ws")
	m.Links["/home/ws/.config/systemd"] = "/srv/target"
	_, err := step.Run(t.Context(), ss, &effecttest.Out{}, false)
	if err == nil || !strings.Contains(err.Error(), "is a link to /srv/target") {
		t.Errorf("a link at the user's systemd directory: %v", err)
	}
}
