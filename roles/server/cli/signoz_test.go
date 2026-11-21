package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestSignozRefusesANonRoot(t *testing.T) {
	for _, cmd := range []string{"password", "sync"} {
		p := fakeProcess()
		p.Euid = 1000
		status, _, stderr := runWs(p, "", "server", "signoz", cmd, "--check")
		if want := "ws server signoz " + cmd + ": run it as root\n"; status != 1 || stderr != want {
			t.Errorf("%s: status = %d, stderr = %q", cmd, status, stderr)
		}
	}
}

// rootProcess is fakeProcess run as root.
func rootProcess() enginecli.Env {
	p := fakeProcess()
	p.Euid = 0
	return p
}

func TestSignozPasswordCheckFailsWhereItWouldWrite(t *testing.T) {
	p := rootProcess()
	m := p.Machine.(*effecttest.Machine)
	m.Dirs["/opt/signoz"] = 0o755
	status, _, stderr := runWs(p, "", "server", "signoz", "password", "--check")
	if status != 1 || stderr != "ws server signoz password: signoz password: generate the root user's password\n"+
		"ws server signoz password: changes are due\n" {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if _, ok := m.Files["/opt/signoz/root.env"]; ok {
		t.Error("--check wrote root.env")
	}
}

func TestSignozPasswordWritesOnce(t *testing.T) {
	p := rootProcess()
	m := p.Machine.(*effecttest.Machine)
	m.Dirs["/opt/signoz"] = 0o755
	if status, _, stderr := runWs(p, "", "server", "signoz", "password"); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	first := string(m.Files["/opt/signoz/root.env"])
	if first == "" || m.Modes["/opt/signoz/root.env"] != 0o600 {
		t.Fatalf("root.env = %q, mode %o; want a password, 600", first, m.Modes["/opt/signoz/root.env"])
	}
	for _, args := range [][]string{{"password", "--check"}, {"password"}} {
		if status, _, stderr := runWs(p, "", append([]string{"server", "signoz"}, args...)...); status != 0 || stderr != "" {
			t.Errorf("%q: status = %d, stderr = %q; want nothing to do", args, status, stderr)
		}
	}
	if string(m.Files["/opt/signoz/root.env"]) != first {
		t.Error("a second run replaced the password")
	}
}

// signozProcess returns a booted machine whose values point at a fake SigNoz. It keeps logs for
// logsDays, and traces and metrics for 90 days. It holds no dashboard, and its migrator is done.
func signozProcess(t *testing.T, logsDays int) enginecli.Env {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/readyz", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("/api/v1/settings/ttl", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"traces_ttl_duration_hrs": 2160, "metrics_ttl_duration_hrs": 2160, "status": "success"}`)
	})
	mux.HandleFunc("/api/v2/settings/ttl", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"default_ttl_days": %d, "status": "success", "ttl_conditions": []}`, logsDays)
	})
	mux.HandleFunc("/api/v2/dashboards", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"data": {"dashboards": [], "total": 0}}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	p := rootProcess()
	m := p.Machine.(*effecttest.Machine)
	m.Dirs["/run/systemd/system"] = 0o755
	m.Files[serverVars] = []byte("WS_SIGNOZ_URL: " + srv.URL + "\nWS_SIGNOZ_DASHBOARDS: []\n")
	m.Results["docker container ls --all --filter name=^signoz-telemetrystore-migrator$ --format {{.Status}}"] =
		effecttest.Result{Out: []byte("Exited (0) 1 hour ago\n")}
	return p
}

func TestSignozSyncCheckPassesWhenInSync(t *testing.T) {
	status, _, stderr := runWs(signozProcess(t, 90), "", "server", "signoz", "sync", "--check")
	if status != 0 || stderr != "" {
		t.Errorf("status = %d, stderr = %q; want nothing to do", status, stderr)
	}
}

func TestSignozSyncCheckFailsWhereTheRetentionDiffers(t *testing.T) {
	status, _, stderr := runWs(signozProcess(t, 15), "", "server", "signoz", "sync", "--check")
	want := "ws server signoz sync: signoz retention: set the retention of logs to 90 days (now logs: 15 days, success)\n" +
		"ws server signoz sync: changes are due\n"
	if status != 1 || stderr != want {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}
