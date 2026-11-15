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

func TestParseKeepsDefaultsBesideAGivenKey(t *testing.T) {
	got, err := Parse([]byte("WS_BASE_SWAP_SIZE_MB: 64\nWS_SECURITY_OUTBOUND: false\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Base.SwapSizeMB == nil || *got.Base.SwapSizeMB != 64 || got.Security.Outbound {
		t.Errorf("given keys not read: %v %v", got.Base.SwapSizeMB, got.Security.Outbound)
	}
	if got.Base.Locale != "en_US.UTF-8" {
		t.Error("a key left out lost its default")
	}
}

func TestParseDerivesFromTheUser(t *testing.T) {
	tests := []struct{ in, home, key string }{
		{"", "/home/zadykian", "/home/zadykian/.ssh/id_ed25519_signing"},
		{"WS_USER: root\n", "/root", "/root/.ssh/id_ed25519_signing"},
		{"WS_USER: dev\n", "/home/dev", "/home/dev/.ssh/id_ed25519_signing"},
		{"WS_HOME: /srv/x\n", "/srv/x", "/srv/x/.ssh/id_ed25519_signing"},
	}
	for _, tt := range tests {
		got, err := Parse([]byte(tt.in))
		if err != nil {
			t.Fatal(err)
		}
		if got.Home != tt.home || got.GitSigningKey != tt.key {
			t.Errorf("Parse(%q) = %q %q", tt.in, got.Home, got.GitSigningKey)
		}
	}
}

func TestParseKeepsAKeyBelowTheHome(t *testing.T) {
	for _, in := range []string{
		"WS_USER: root\nWS_GIT_SIGNING_KEY: /root/k\n",
		"WS_HOME: /srv/x\nWS_GIT_SIGNING_KEY: /srv/x/a/../b/k\n",
	} {
		if _, err := Parse([]byte(in)); err != nil {
			t.Errorf("Parse(%q): %v", in, err)
		}
	}
}

func TestParseRefusesAKeyOutsideTheHome(t *testing.T) {
	tests := map[string]string{
		"outside":          "WS_GIT_SIGNING_KEY: /etc/key\n",
		"dot dot":          "WS_GIT_SIGNING_KEY: /home/zadykian/../root/key\n",
		"the home":         "WS_GIT_SIGNING_KEY: /home/zadykian\n",
		"a sibling":        "WS_GIT_SIGNING_KEY: /home/zadykian2/key\n",
		"relative":         "WS_GIT_SIGNING_KEY: key\n",
		"outside, root":    "WS_USER: root\nWS_GIT_SIGNING_KEY: /tmp/key\n",
		"outside, WS_HOME": "WS_HOME: /srv/x\nWS_GIT_SIGNING_KEY: /home/zadykian/key\n",
	}
	for name, in := range tests {
		_, err := Parse([]byte(in))
		if err == nil || !strings.Contains(err.Error(), "is outside WS_HOME") {
			t.Errorf("%s: Parse(%q) = %v", name, in, err)
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
