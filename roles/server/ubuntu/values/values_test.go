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
	for in, want := range map[string]int{"": 65536, "zed_nofile: 4096\n": 4096} {
		got, err := Parse([]byte(in))
		if err != nil || got.Zed.Nofile != want {
			t.Errorf("Parse(%q) = %d, %v; want %d", in, got.Zed.Nofile, err, want)
		}
	}
}

func TestParseKeepsDefaultsBesideAGivenKey(t *testing.T) {
	got, err := Parse([]byte("base_swap_size_mb: 64\nsecurity_outbound: false\n"))
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
		{"ws_user: root\n", "/root", "/root/.ssh/id_ed25519_signing"},
		{"ws_user: dev\n", "/home/dev", "/home/dev/.ssh/id_ed25519_signing"},
		{"ws_home: /srv/x\n", "/srv/x", "/srv/x/.ssh/id_ed25519_signing"},
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
		"ws_user: root\nws_git_signing_key: /root/k\n",
		"ws_home: /srv/x\nshell_git_allowed_signers: /srv/x/a/../b/signers\n",
	} {
		if _, err := Parse([]byte(in)); err != nil {
			t.Errorf("Parse(%q): %v", in, err)
		}
	}
}

func TestParseRefusesAKeyOutsideTheHome(t *testing.T) {
	tests := map[string]string{
		"outside":          "ws_git_signing_key: /etc/key\n",
		"dot dot":          "ws_git_signing_key: /home/zadykian/../root/key\n",
		"the home":         "ws_git_signing_key: /home/zadykian\n",
		"a sibling":        "shell_git_allowed_signers: /home/zadykian2/signers\n",
		"relative":         "shell_git_allowed_signers: signers\n",
		"outside, root":    "ws_user: root\nws_git_signing_key: /tmp/key\n",
		"outside, ws_home": "ws_home: /srv/x\nws_git_signing_key: /home/zadykian/key\n",
	}
	for name, in := range tests {
		_, err := Parse([]byte(in))
		if err == nil || !strings.Contains(err.Error(), "is outside ws_home") {
			t.Errorf("%s: Parse(%q) = %v", name, in, err)
		}
	}
}

func TestParseRefuses(t *testing.T) {
	tests := map[string]string{
		"unknown key":   "nonsense: 1\n",
		"duplicate key": "base_locale: a\nbase_locale: b\n",
		"wrong type":    "base_swappiness: lots\n",
		"unknown field": "wireguard_peers:\n  - name: a\n    key: b\n",
		"not a mapping": "- a\n",
		"a secret key":  "shell_secret_env:\n  TOKEN: hunter2\n",
		"a password":    "ikeward_p12_password: hunter2\n",
	}
	for name, in := range tests {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("%s: Parse(%q) found no error", name, in)
		} else if strings.Contains(err.Error(), "hunter2") {
			t.Errorf("%s: the error shows the value: %v", name, err)
		}
	}
}
