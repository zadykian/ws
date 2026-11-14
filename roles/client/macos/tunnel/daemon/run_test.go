package daemon

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/tunnel/daemon/uapi"
)

func TestRunSetsUpTheUtun(t *testing.T) {
	r := newRig("")
	if err := r.play(); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"rm -f /var/run/ws-tunnel.state /var/run/ws-tunnel.state.new",
		"ls -1 /etc/ws-tunnel/routes.d",
		"ifconfig utun4 inet 10.99.0.2/32 10.99.0.2 alias",
		"ifconfig utun4 up",
		"route -q -n add -inet 10.99.0.1/32 -interface utun4",
		"ifconfig",
		"mv -f /var/run/ws-tunnel.state.new /var/run/ws-tunnel.state",
		"rm -f /var/run/ws-tunnel.state /var/run/ws-tunnel.state.new",
	}
	if !slices.Equal(r.m.runs, want) {
		t.Errorf("commands = %q, want %q", r.m.runs, want)
	}
	if key, _ := uapi.SetPrivateKey(macKey); !slices.Equal(r.dev.sets, []string{key}) {
		t.Errorf("device got %q, want the Mac's key alone", r.dev.sets)
	}
	if got, want := r.lines(), []string{
		"started on utun4, 10.99.0.2", "no path to ws answers, as the Mac is on none; trying again 10 s after each round",
		"stopped",
	}; !slices.Equal(got, want) {
		t.Errorf("log = %q, want %q", got, want)
	}
}

func TestRunSetsUpTheUtunAtTheSlot(t *testing.T) {
	r := newRig("WS_SLOT=1\n")
	if err := r.play(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"ifconfig utun4 inet 10.99.1.2/32 10.99.1.2 alias",
		"route -q -n add -inet 10.99.1.1/32 -interface utun4",
	} {
		if !slices.Contains(r.m.runs, want) {
			t.Errorf("commands = %q, want %q", r.m.runs, want)
		}
	}
	if got := r.lines()[0]; got != "started on utun4, 10.99.1.2" {
		t.Errorf("log = %q", r.lines())
	}
}

func TestRunFailsToStart(t *testing.T) {
	tests := []struct {
		name  string
		spoil func(*rig)
		want  string
	}{
		{"no conf", func(r *rig) { delete(r.m.Files, dir+"/tunnel.conf") }, "cannot read /etc/ws-tunnel/tunnel.conf"},
		{"no key", func(r *rig) { delete(r.m.Files, dir+"/private.key") }, "cannot read /etc/ws-tunnel/private.key"},
		{"no address", func(r *rig) {
			r.m.table["ifconfig utun4 up"] = effecttest.Result{Err: errTest}
		}, "cannot set up utun4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig("")
			tt.spoil(r)
			err := r.play()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want %q", err, tt.want)
			}
			if got := r.log.String(); !strings.Contains(got, tt.want) || strings.Contains(got, "stopped") {
				t.Errorf("log = %q, want the error and no stop", got)
			}
		})
	}
}

func TestRunExits(t *testing.T) {
	r := newRig("")
	if err := r.play(Exited); err == nil || !strings.Contains(r.log.String(), "wireguard-go has exited") {
		t.Errorf("err = %v, log = %q", err, r.log.String())
	}
}

func TestRunStopsOnceANewWsTakesItsPlace(t *testing.T) {
	dir := t.TempDir()
	old, fresh := filepath.Join(dir, "old"), filepath.Join(dir, "new")
	for _, p := range []string{old, fresh} {
		if err := os.WriteFile(p, []byte("ws"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	const line = "root's copy of ws is new; stops, for launchd to start the new one"
	for _, tt := range []struct {
		name, later string
		stops       bool
	}{
		{"a new file", fresh, true},
		{"the same file", old, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig("")
			file := old
			r.Program = func() (fs.FileInfo, error) {
				info, err := os.Stat(file)
				file = tt.later
				return info, err
			}
			if err := r.play(Tick); err != nil {
				t.Fatal(err)
			}
			if got := slices.Contains(r.lines(), line); got != tt.stops {
				t.Errorf("log = %q, want the stop %v", r.lines(), tt.stops)
			}
		})
	}
}
