package daemon

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/uapi"
)

func TestRunSetsUpTheUtun(t *testing.T) {
	r := newRig("")
	if err := r.play(); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"ls -1 /etc/ws-tunnel/routes.d",
		"ifconfig utun4 inet 10.99.0.2/32 10.99.0.2 alias",
		"ifconfig utun4 up",
		"route -q -n add -inet 10.99.0.1/32 -interface utun4",
		"rm -f /var/run/ws-tunnel.state /var/run/ws-tunnel.state.new",
	}
	if !slices.Equal(r.m.runs, want) {
		t.Errorf("commands = %q, want %q", r.m.runs, want)
	}
	if key, _ := uapi.SetPrivateKey(macKey); !slices.Equal(r.dev.sets, []string{key}) {
		t.Errorf("device got %q, want the Mac's key alone", r.dev.sets)
	}
	if got, want := r.lines(), []string{"started on utun4, 10.99.0.2", "stopped"}; !slices.Equal(got, want) {
		t.Errorf("log = %q, want %q", got, want)
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
