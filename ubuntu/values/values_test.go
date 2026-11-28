package values

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseEmptyGivesDefault(t *testing.T) {
	for _, in := range []string{"", "---\n", "# nothing\n"} {
		got, err := Parse([]byte(in))
		if err != nil {
			t.Fatalf("Parse(%q): %v", in, err)
		}
		want := Default()
		want.derive()
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Parse(%q) differs from Default", in)
		}
	}
}

func TestParseReadsZedNofile(t *testing.T) {
	for in, want := range map[string]int{"": 65536, "WS_ZED_NOFILE: 4096\n": 4096} {
		got, err := Parse([]byte(in))
		if err != nil || got.Zed.Nofile != want {
			t.Errorf("Parse(%q) = %d, %v; want %d", in, got.Zed.Nofile, err, want)
		}
	}
}

func TestParseReadsTheGPUSwitch(t *testing.T) {
	for in, want := range map[string]bool{"": false, "WS_NVIDIA_GPU: true\n": true, "WS_NVIDIA_GPU: false\n": false} {
		got, err := Parse([]byte(in))
		if err != nil || got.Nvidia.GPU != want {
			t.Errorf("Parse(%q) = %t, %v; want %t", in, got.Nvidia.GPU, err, want)
		}
	}
}

func TestParseKeepsDefaultsBesideAGivenKey(t *testing.T) {
	got, err := Parse([]byte("WS_BASE_SWAP_SIZE_MB: 64\nWS_BASE_SWAPPINESS: 20\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Base.SwapSizeMB == nil || *got.Base.SwapSizeMB != 64 || got.Base.Swappiness != 20 {
		t.Errorf("given keys not read: %v %v", got.Base.SwapSizeMB, got.Base.Swappiness)
	}
	if got.Base.Locale != "en_US.UTF-8" {
		t.Error("a key left out lost its default")
	}
}

func TestParseRefusesAnOldKey(t *testing.T) {
	for _, old := range []string{
		"docker_log_max_size: 1m", "ws_user: ada", "WS_USER: root", "WS_HOME: /root", "wireguard_peers: []",
		"network_direct_link_interface: enp5s0", "WS_SHELL_GIT_CLONES: []", "WS_GIT_SIGNING_PUB_KEY: /root/.ssh/k.pub",
	} {
		key, _, _ := strings.Cut(old, ":")
		if _, err := Parse([]byte(old + "\n")); err == nil || !strings.Contains(err.Error(), key) {
			t.Errorf("Parse(%q) = %v, want an error that names %s", old, err, key)
		}
	}
}

func TestParseRefuses(t *testing.T) {
	tests := map[string]string{
		"unknown key":    "nonsense: 1\n",
		"a dropped key":  "shell_git_signing: false\n",
		"a dropped name": "shell_git_name: a\n",
		"a dropped path": "shell_git_allowed_signers: /etc/ws/allowed_signers\n",
		"duplicate key":  "WS_BASE_LOCALE: a\nWS_BASE_LOCALE: b\n",
		"wrong type":     "WS_BASE_SWAPPINESS: lots\n",
		"unknown field":  "WS_TUNNEL_PEERS:\n  - name: a\n    key: b\n",
		"not a mapping":  "- a\n",
		"a secret key":   "shell_secret_env:\n  TOKEN: hunter2\n",
		"a password":     "ikeward_p12_password: hunter2\n",
	}
	for name, in := range tests {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("%s: Parse(%q) found no error", name, in)
		} else if strings.Contains(err.Error(), "hunter2") {
			t.Errorf("%s: the error shows the value: %v", name, err)
		}
	}
}
