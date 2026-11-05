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

func policy(t *testing.T, j map[string]any) []any {
	gc := j["builder"].(map[string]any)["gc"].(map[string]any)
	if gc["enabled"] != true {
		t.Errorf("gc = %v, want enabled", gc)
	}
	return gc["policy"].([]any)
}

func TestDaemonJSONPolicyHasTheAgeInHours(t *testing.T) {
	p := policy(t, daemon(t, 90))
	if len(p) != 3 {
		t.Fatalf("policy = %v, want an age rule and two limits", p)
	}
	if age := p[0].(map[string]any); age["all"] != true || age["filter"].([]any)[0] != "until=2160h" {
		t.Errorf("first rule = %v, want all with until=2160h", age)
	}
	if p[2].(map[string]any)["maxUsedSpace"] != "9GB" {
		t.Errorf("last rule = %v, want the limits", p[2])
	}
}

func TestDaemonJSONLeavesOutTheAgeOfZeroDays(t *testing.T) {
	if p := policy(t, daemon(t, 0)); len(p) != 2 {
		t.Errorf("policy = %v, want the two limits alone", p)
	}
}

func TestRootlessJSONHasTheUsersValuesAndNoSystemOnes(t *testing.T) {
	d := values.Default().Docker
	out, err := docker.RootlessJSON(d, docker.Cache{Reserved: "2GB", MaxUsed: "9GB", MinFree: "3GB"})
	if err != nil {
		t.Fatal(err)
	}
	var j map[string]any
	if err := json.Unmarshal(out, &j); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{"cgroup-parent": "docker.slice", "ip": "127.0.0.1", "log-driver": "json-file"} {
		if j[key] != want {
			t.Errorf("%s = %v, want %v", key, j[key], want)
		}
	}
	if dns := j["dns"].([]any); len(dns) != 1 || dns[0] != "10.0.2.3" {
		t.Errorf("dns = %v, want slirp4netns's resolver", dns)
	}
	for _, key := range []string{"bip", "default-address-pools", "live-restore"} {
		if _, ok := j[key]; ok {
			t.Errorf("%s is set; the rootless daemon has no use for it", key)
		}
	}
}
