package docker_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/server/docker"
	"github.com/zadykian/ws/internal/server/values"
)

func daemon(t *testing.T, days int) map[string]any {
	d := values.Default().Docker
	d.BuildCacheKeepDays = days
	out, err := docker.DaemonJSON(d, docker.Cache{Reserved: "2GB", MaxUsed: "9GB", MinFree: "3GB"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(out), "}\n") {
		t.Errorf("daemon.json = %q, want a final newline", out)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	return got
}

func TestDaemonJSONHasTheDefaultValues(t *testing.T) {
	j := daemon(t, 90)
	for key, want := range map[string]any{"bip": "172.17.0.1/16", "ip": "127.0.0.1", "cgroup-parent": "docker.slice",
		"live-restore": true, "log-driver": "json-file"} {
		if j[key] != want {
			t.Errorf("%s = %v, want %v", key, j[key], want)
		}
	}
	if dns := j["dns"].([]any); len(dns) != 1 || dns[0] != "172.17.0.1" {
		t.Errorf("dns = %v, want the bridge's address", dns)
	}
	opts := j["log-opts"].(map[string]any)
	if opts["max-size"] != "10m" || opts["max-file"] != "3" {
		t.Errorf("log-opts = %v, want 10m and 3", opts)
	}
	pool := j["default-address-pools"].([]any)[0].(map[string]any)
	if pool["base"] != "172.16.0.0/12" || pool["size"] != float64(20) {
		t.Errorf("pool = %v, want 172.16.0.0/12 and /20", pool)
	}
}
