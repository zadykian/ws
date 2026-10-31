package daemon

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

const (
	dir  = "/etc/ws-tunnel"
	conf = "WS_PUBLIC_KEY=AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=\nWS_REMOTE=ws.example.org\n"
)

func machine(files map[string]string) *effecttest.Machine {
	m := effecttest.NewMachine()
	for name, text := range files {
		m.Files[name] = []byte(text)
	}
	return m
}

func TestReadSettings(t *testing.T) {
	m := machine(map[string]string{
		dir + "/tunnel.conf":    conf,
		dir + "/routes.d/a":     "192.168.50.0/24\n10.99.0.0/25\n",
		dir + "/routes.d/b":     "# kept apart\n192.168.50.0/24\n172.16.0.0/12\n",
		dir + "/routes.d/other": "198.51.100.0/24\n",
	})
	m.Results["ls -1 "+dir+"/routes.d"] = effecttest.Result{Out: []byte("a\nb\nsub\n")}
	got, err := ReadSettings(context.Background(), m, dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []netip.Prefix{netip.MustParsePrefix("192.168.50.0/24"), netip.MustParsePrefix("172.16.0.0/12")}
	if len(got.Routes) != 2 || got.Routes[0] != want[0] || got.Routes[1] != want[1] {
		t.Errorf("Routes = %v, want %v", got.Routes, want)
	}
	if len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "routes.d/a: leaves out '10.99.0.0/25'") {
		t.Errorf("Warnings = %q", got.Warnings)
	}
	if got.Conf.Remote != "ws.example.org" {
		t.Errorf("Conf = %+v", got.Conf)
	}
}

func TestReadSettingsNoRoutes(t *testing.T) {
	m := machine(map[string]string{dir + "/tunnel.conf": conf})
	m.Results["ls -1 "+dir+"/routes.d"] = effecttest.Result{Err: errors.New("no such directory")}
	got, err := ReadSettings(context.Background(), m, dir)
	if err != nil || len(got.Routes) != 0 {
		t.Errorf("got %+v, %v; want no routes", got, err)
	}
}

func TestReadSettingsFails(t *testing.T) {
	tests := []struct{ name, conf, want string }{
		{"missing", "", "cannot read /etc/ws-tunnel/tunnel.conf: run mac/setup.sh"},
		{"bad line", "WS_PORT=0\n", "tunnel.conf: cannot take 'WS_PORT=0'"},
		{"no key", "WS_REMOTE=a.b\n", "no WS_PUBLIC_KEY"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := machine(nil)
			if tt.conf != "" {
				m.Files[dir+"/tunnel.conf"] = []byte(tt.conf)
			}
			m.Files[dir+"/routes.d/a"] = []byte("192.168.50.0/24\n")
			m.Results["ls -1 "+dir+"/routes.d"] = effecttest.Result{Out: []byte("a\n")}
			got, err := ReadSettings(context.Background(), m, dir)
			if len(got.Routes) != 1 {
				t.Errorf("Routes = %v, want the route kept", got.Routes)
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestReadKey(t *testing.T) {
	good := "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="
	m := machine(map[string]string{dir + "/private.key": good + "\n"})
	if got, err := ReadKey(m, dir); err != nil || got != good {
		t.Errorf("ReadKey = %q, %v", got, err)
	}
	m.Files[dir+"/private.key"] = []byte("short")
	if _, err := ReadKey(m, dir); err == nil {
		t.Error("ReadKey took a short key")
	}
	if _, err := ReadKey(machine(nil), dir); err == nil {
		t.Error("ReadKey took a missing file")
	}
}
