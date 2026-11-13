package settings

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const (
	dir  = "/etc/ws-tunnel"
	conf = "WS_PUBLIC_KEY=AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=\nWS_REMOTE=ws.example.org\n"
)

// paths are the files of the daemon under dir.
var paths = Paths{
	Conf: dir + "/tunnel.conf", Key: dir + "/private.key", Routes: dir + "/routes.d",
	State: "/var/run/ws-tunnel.state", Log: "/var/log/ws-tunnel.log",
}

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
	got, err := ReadSettings(context.Background(), m, paths)
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
	got, err := ReadSettings(context.Background(), m, paths)
	if err != nil || len(got.Routes) != 0 {
		t.Errorf("got %+v, %v; want no routes", got, err)
	}
}

func TestReadSettingsFails(t *testing.T) {
	tests := []struct{ name, conf, want string }{
		{"missing", "", "cannot read /etc/ws-tunnel/tunnel.conf: run ws client macos apply"},
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
			got, err := ReadSettings(context.Background(), m, paths)
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
	if got, err := ReadKey(m, paths.Key); err != nil || got != good {
		t.Errorf("ReadKey = %q, %v", got, err)
	}
	m.Files[dir+"/private.key"] = []byte("short")
	if _, err := ReadKey(m, paths.Key); err == nil {
		t.Error("ReadKey took a short key")
	}
	if _, err := ReadKey(machine(nil), paths.Key); err == nil {
		t.Error("ReadKey took a missing file")
	}
}

func TestReadSettingsOthers(t *testing.T) {
	p := paths
	p.Conf, p.Routes, p.Root = dir+"/ws/tunnel.conf", dir+"/ws/routes.d", dir
	m := machine(map[string]string{
		p.Conf:                   conf,
		dir + "/lab/tunnel.conf": "WS_PUBLIC_KEY=AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=\nWS_REMOTE=lab.example.org\n",
		dir + "/lab/routes.d/a":  "10.50.0.0/16\n",
		dir + "/bad/tunnel.conf": "WS_PORT=0\n",
		dir + "/bad/routes.d/a":  "not a network\n10.60.0.0/16\n",
		dir + "/gone/routes.d/a": "10.70.0.0/16\n",
		dir + "/private.key":     "key",
	})
	m.Results["ls -1p "+dir] = effecttest.Result{Out: []byte("bad/\ngone/\nlab/\nprivate.key\nws/\n")}
	m.Results["ls -1 "+dir+"/lab/routes.d"] = effecttest.Result{Out: []byte("a\n")}
	m.Results["ls -1 "+dir+"/bad/routes.d"] = effecttest.Result{Out: []byte("a\n")}
	m.Results["ls -1 "+dir+"/gone/routes.d"] = effecttest.Result{Out: []byte("a\n")}
	got, err := ReadSettings(context.Background(), m, p)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, o := range got.Others {
		names = append(names, o.Name)
	}
	if strings.Join(names, " ") != "bad gone lab" {
		t.Fatalf("Others = %v, want bad gone lab: not its own, not private.key", names)
	}
	bad, gone, lab := got.Others[0], got.Others[1], got.Others[2]
	if lab.Conf.Remote != "lab.example.org" || len(lab.Routes) != 1 || lab.Routes[0] != netip.MustParsePrefix("10.50.0.0/16") {
		t.Errorf("lab = %+v", lab)
	}
	if gone.Conf.Key != "" || len(gone.Routes) != 1 {
		t.Errorf("gone = %+v, want no conf and its routes", gone)
	}
	if bad.Conf.Key != "" || len(bad.Routes) != 1 {
		t.Errorf("bad = %+v, want no conf and its valid route", bad)
	}
	if len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "leaves out server bad's endpoints") {
		t.Errorf("Warnings = %q, want bad's endpoints alone", got.Warnings)
	}
}
