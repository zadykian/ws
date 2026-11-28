package tunnel_test

import (
	"context"
	"errors"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/tunnel"
)

var yielded = []netip.Prefix{netip.MustParsePrefix("10.20.0.0/16"), netip.MustParsePrefix("192.168.5.0/24")}

// yieldMachine has the daemons of ws and lab loaded and running, with states that show no yield,
// and old's plist on disk with no job loaded.
func yieldMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs["/Library/LaunchDaemons"] = 0o755
	for _, n := range []string{"ws", "lab", "old"} {
		m.Files["/Library/LaunchDaemons/"+tunnel.For(n).Label+".plist"] = []byte("x")
	}
	for _, n := range []string{"ws", "lab"} {
		label := tunnel.For(n).Label
		m.Results["launchctl print system/"+label] = effecttest.Result{}
		m.Results["launchctl kill SIGHUP system/"+label] = effecttest.Result{}
		m.Files[tunnel.For(n).State] = []byte(stateWith(""))
	}
	return m
}

func stateWith(yield string) string {
	return "tier=lan\nendpoint=192.0.2.7:51820\ninterface=utun4\nsince=0\nroutes=\nleft_out=\nyielded=" + yield + "\n"
}

// takes returns a Sleep that has each daemon of names show yield at the first pause, and counts the
// pauses.
func takes(m *effecttest.Machine, yield string, pauses *int, names ...string) func(time.Duration) {
	return func(time.Duration) {
		*pauses++
		for _, n := range names {
			m.Files[tunnel.For(n).State] = []byte(stateWith(yield))
		}
	}
}

func TestYieldWritesTheFileSignalsEachLoadedDaemonAndWaits(t *testing.T) {
	m := yieldMachine()
	pauses := 0
	y := tunnel.Yielder{M: m, Sleep: takes(m, "10.20.0.0/16 192.168.5.0/24", &pauses, "ws", "lab")}
	if err := y.Yield(context.Background(), append(yielded, yielded[0])); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[tunnel.YieldFile]); got != "10.20.0.0/16\n192.168.5.0/24\n" {
		t.Errorf("the yield = %q, want each network once, a line each", got)
	}
	if _, ok := m.Root[tunnel.YieldFile]; !ok {
		t.Errorf("the yield is not root's")
	}
	write := slices.Index(m.Calls, "write root "+tunnel.YieldFile)
	for _, n := range []string{"ws", "lab"} {
		if kill := slices.Index(m.Calls, "root launchctl kill SIGHUP system/"+tunnel.For(n).Label); kill < write {
			t.Errorf("did not signal %s after the write: calls = %q", n, m.Calls)
		}
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "kill SIGHUP system/"+tunnel.For("old").Label) }) {
		t.Errorf("signalled a daemon that launchd has not loaded: calls = %q", m.Calls)
	}
	if pauses != 1 {
		t.Errorf("paused %d times, want once, until the states show the yield", pauses)
	}
}

func TestYieldOfNoNetworkRemovesTheFileAndWaitsForTheTakeBack(t *testing.T) {
	m := yieldMachine()
	m.Files[tunnel.YieldFile] = []byte("10.20.0.0/16\n")
	for _, n := range []string{"ws", "lab"} {
		m.Files[tunnel.For(n).State] = []byte(stateWith("10.20.0.0/16"))
	}
	pauses := 0
	if err := (tunnel.Yielder{M: m, Sleep: takes(m, "", &pauses, "ws", "lab")}).Yield(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files[tunnel.YieldFile]; ok || !slices.Contains(m.Calls, "root rm -f "+tunnel.YieldFile) {
		t.Errorf("kept the yield: calls = %q", m.Calls)
	}
	if pauses != 1 {
		t.Errorf("paused %d times, want once, until the states show no yield", pauses)
	}
}

func TestYieldFailsOnAFailedSignal(t *testing.T) {
	m := yieldMachine()
	m.Results["launchctl kill SIGHUP system/"+tunnel.For("lab").Label] = effecttest.Result{Err: errors.New("exit status 3")}
	pauses := 0
	err := tunnel.Yielder{M: m, Sleep: takes(m, "10.20.0.0/16", &pauses, "ws")}.Yield(context.Background(), yielded[:1])
	if err == nil || !strings.Contains(err.Error(), "cannot signal system/"+tunnel.For("lab").Label) {
		t.Errorf("err = %v, want the failed signal named", err)
	}
	if strings.Contains(err.Error(), "the daemon of ws") {
		t.Errorf("err = %v, want ws's daemon, which took the yield, left out", err)
	}
}

func TestYieldFailsAfterFiveSecondsAndNamesTheDaemon(t *testing.T) {
	// ws's state is readable and still shows no yield: the usual daemon that has not taken it.
	m := yieldMachine()
	pauses := 0
	err := tunnel.Yielder{M: m, Sleep: takes(m, "10.20.0.0/16", &pauses, "lab")}.Yield(context.Background(), yielded[:1])
	if err == nil || !strings.Contains(err.Error(), "the daemon of ws has not taken the yield after 5s; its log is "+tunnel.For("ws").Log) {
		t.Errorf("err = %v, want ws's daemon named", err)
	}
	if err != nil && strings.Contains(err.Error(), "daemon of lab") {
		t.Errorf("err = %v, want lab's daemon, which took the yield, left out", err)
	}
	if want := int(tunnel.YieldWait / (100 * time.Millisecond)); pauses != want {
		t.Errorf("paused %d times, want %d, 5 s in all", pauses, want)
	}
}
