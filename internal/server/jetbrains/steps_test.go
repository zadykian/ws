package jetbrains

import (
	"context"
	"errors"
	"os/user"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

const goList = "curl -fsSL https://data.services.jetbrains.com/products/releases?code=GO&latest=true&type=release"

func lookupUser(name string) (*user.User, error) {
	if name != "u" {
		return nil, errors.New("unknown user " + name)
	}
	return &user.User{Uid: "1000", Gid: "1001"}, nil
}

func stepFixture(codes ...string) (*effecttest.Machine, []step.Step) {
	m, s, _ := fixture(1 << 30)
	for _, d := range []string{"/h/.cache", "/h/.cache/JetBrains", "/h/.cache/JetBrains/RemoteDev"} {
		m.Dirs[d] = 0o755
	}
	v := values.Default()
	v.User, v.Home = "u", "/h"
	v.JetBrains.Backends = codes
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	return m, steps(v, s, &effecttest.Units{}, "/usr/local/bin/ws", "", lookupUser)
}

func names(ss []step.Step) string {
	var n []string
	for _, s := range ss {
		n = append(n, s.Name())
	}
	return strings.Join(n, ",")
}

func TestStepsOrderTheIcuAndTheTimerBeforeTheBackends(t *testing.T) {
	_, ss := stepFixture("RD", "GO")
	if got := names(ss); got != "jetbrains ICU,jetbrains timer,jetbrains backends" {
		t.Errorf("steps %s", got)
	}
	if ss[1].WaitsOn() != icuStep || ss[2].WaitsOn() != timerStep {
		t.Errorf("waits on %q, %q", ss[1].WaitsOn(), ss[2].WaitsOn())
	}
}

func TestStepsNeedNoIcuWithoutRider(t *testing.T) {
	_, ss := stepFixture("GO")
	if got := names(ss); got != "jetbrains timer,jetbrains backends" || ss[0].WaitsOn() != "" {
		t.Errorf("steps %s", got)
	}
}

func TestBackendsStepInstallsTheLatestBuild(t *testing.T) {
	m, ss := stepFixture("GO")
	m.Results[goList] = effecttest.Result{Out: []byte(list)}
	m.Results["curl -fsSL "+sumLink] = effecttest.Result{Out: []byte(strings.Repeat("ab", 32))}
	b, err := ParseReleases([]byte(list), []string{"GO"}, "linux", testDist)
	if err != nil {
		t.Fatal(err)
	}
	notRunning(t, m, b[0].Dir)
	plan, err := ss[1].Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	last := plan.Changes[len(plan.Changes)-1].String()
	if !strings.Contains(last, "would be installed in "+b[0].Dir) {
		t.Errorf("changes %v", plan.Changes)
	}
}

func TestBackendsStepWithNoneRemovesTheStaleDownloadsAlone(t *testing.T) {
	m, ss := stepFixture()
	m.Dirs["/h/.cache/ws-jetbrains/apply"] = 0o755
	plan, err := ss[1].Plan(context.Background())
	if err != nil || len(plan.Changes) != 1 {
		t.Fatalf("%v, %v", plan.Changes, err)
	}
	if strings.Contains(strings.Join(m.Calls, "\n"), "curl") {
		t.Error("read the releases for no backend")
	}
}

func TestBackendsStepRefusesAnArchitectureWithNoDownload(t *testing.T) {
	m, ss := stepFixture("GO")
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("riscv64\n")}
	m.Results[goList] = effecttest.Result{Out: []byte(list)}
	if _, err := ss[1].Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "riscv64") {
		t.Errorf("got %v", err)
	}
}
