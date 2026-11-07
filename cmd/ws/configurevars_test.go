package main

import (
	"context"
	"encoding/base64"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
)

// lockCalls counts the calls of the wizard's lock that hold what.
func lockCalls(calls []string, what string) int {
	return len(slices.DeleteFunc(slices.Clone(calls), func(call string) bool {
		return !strings.Contains(call, what) || !strings.Contains(call, wizard.LockName)
	}))
}

func TestVarsReachTakesTheLockOnceAndLetsItGo(t *testing.T) {
	p, m := macTerminal()
	r := &varsReach{p: p}
	for range 2 {
		if _, err := r.reach(context.Background(), nil, nil); err == nil || !strings.Contains(err.Error(), "no direct way") {
			t.Fatalf("err = %v", err)
		}
	}
	if got := lockCalls(m.Calls, "mkdir"); got != 1 {
		t.Errorf("took the lock %d times", got)
	}
	r.close(context.Background())
	if got := lockCalls(m.Calls, "rm -rf"); got != 1 {
		t.Errorf("let it go %d times: %v", got, m.Calls)
	}
}

func TestVarsReachLeavesAnotherRunsLock(t *testing.T) {
	p, m := macTerminal()
	m.Dirs["/tmp/"+wizard.LockName] = 0o700
	r := &varsReach{p: p}
	if _, err := r.reach(context.Background(), nil, nil); err == nil || !strings.Contains(err.Error(), "another run holds") {
		t.Fatalf("err = %v", err)
	}
	r.close(context.Background())
	if got := lockCalls(m.Calls, "rm -rf"); got != 0 {
		t.Errorf("removed another run's lock: %v", m.Calls)
	}
}

// farWs is a ws that takes the login as root, and whose files script holds file. Each operation of
// the script that reaches it goes on ops, past any that a slow test leaves there.
type farWs struct {
	*effecttest.Machine
	file string
	ops  chan []string
}

func newFarWs(m *effecttest.Machine, file string) farWs {
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\nWS_REMOTE=r.example\nWS_REMOTE_PORT=22\n")
	return farWs{Machine: m, file: file, ops: make(chan []string, 64)}
}

func (w farWs) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	switch {
	case name == "ifconfig":
		return nil, nil
	case name == "ssh" && slices.Contains(args, "-G"):
		return []byte("user root\nhostname ws\nport 22\n"), nil
	}
	return w.Machine.Run(ctx, name, args...)
}

func (w farWs) RunInput(ctx context.Context, in []byte, name string, args ...string) ([]byte, error) {
	at := slices.Index(args, "--")
	if name != "ssh" || at < 0 {
		return w.Machine.RunInput(ctx, in, name, args...)
	}
	w.ops <- args[at+1:]
	return []byte(w.file), nil
}

// A way to root that the scan read holds across a reach, and one it never read is read there.
func TestVarsReachKeepsTheSudoItIsGiven(t *testing.T) {
	p, m := macTerminal()
	p.machine = newFarWs(m, "")
	sudo := &wizard.Sudo{}
	r := &varsReach{p: p, sudo: sudo}
	defer r.close(context.Background())
	got, err := r.reach(context.Background(), nil, []byte("script"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Shell.(wizard.SSHShell).Sudo != sudo || string(got.Script) != "script" {
		t.Errorf("remote = %+v", got)
	}
	if _, err := got.Do(context.Background(), "status"); err != nil {
		t.Errorf("the reach left sudo unread: %v", err)
	}
}

func TestVarsSheetOpensWithWhatWsHolds(t *testing.T) {
	p, m := macTerminal()
	ws := newFarWs(m, "ws_user: ada\n")
	p.machine = ws
	r := &varsReach{p: p}
	defer r.close(context.Background())
	sheet, err := r.open(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	specs := sheet.Fields(nil)
	user := slices.IndexFunc(specs, func(s tui.FieldSpec) bool { return s.Label == "ws_user" })
	outbound := slices.IndexFunc(specs, func(s tui.FieldSpec) bool { return s.Label == "security_outbound" })
	if user < 0 || specs[user].Value != "ada" || specs[user].Default != "" {
		t.Errorf("user at %d of %+v", user, specs)
	}
	if outbound < 0 || specs[outbound].Kind != tui.ChoiceField {
		t.Errorf("outbound at %d of %+v", outbound, specs)
	}
	if op := <-ws.ops; !slices.Equal(op, []string{"values"}) {
		t.Errorf("op = %q", op)
	}
}

func TestMacConfigureSavesHostVarsAndLetsTheLockGo(t *testing.T) {
	p, m := macTerminal()
	ws := newFarWs(m, "ws_user: ada\n")
	p.machine = ws
	in, keys := io.Pipe()
	defer func() { _ = keys.Close() }()
	ended := make(chan int, 1)
	go func() {
		root := newRoot(p, newMacCmd(p), newServerCmd(p))
		ended <- execute(context.Background(), root, []string{"mac", "configure"}, in, io.Discard, io.Discard)
	}()
	press := func(text string) {
		if _, err := keys.Write([]byte(text)); err != nil {
			t.Error(err)
		}
	}
	// The secrets sheet's read holds the keys, so Ctrl-N goes on until host_vars' read lands.
	for read := false; !read; {
		select {
		case op := <-ws.ops:
			if read = true; !slices.Equal(op, []string{"values"}) {
				t.Fatalf("read = %q", op)
			}
		case <-time.After(150 * time.Millisecond):
			press("\x0e")
		}
	}
	// A read holds the keys, so the edit goes on until one lands past it.
	var saved []string
	for saved == nil {
		select {
		case op := <-ws.ops:
			if len(op) > 1 {
				saved = op
			}
		case <-time.After(150 * time.Millisecond):
			press("x\x13")
		}
	}
	pair, err := base64.StdEncoding.DecodeString(saved[1])
	if err != nil || !strings.HasPrefix(string(pair), "ws_user=adax") {
		t.Errorf("saved %q: %v", pair, err)
	}
	for {
		select {
		case status := <-ended:
			if got := lockCalls(m.Calls, "rm -rf"); status != 0 || got != 1 {
				t.Errorf("status = %d, let the lock go %d times", status, got)
			}
			return
		case <-time.After(150 * time.Millisecond):
			press("\x1b")
		}
	}
}
