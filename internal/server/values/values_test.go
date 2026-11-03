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

func TestParseKeepsDefaultsBesideAGivenKey(t *testing.T) {
	got, err := Parse([]byte("jetbrains_backends: []\nbase_swap_size_mb: 64\nsecurity_outbound: false\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.JetBrains.Backends) != 0 || got.Base.SwapSizeMB == nil || *got.Base.SwapSizeMB != 64 || got.Security.Outbound {
		t.Errorf("given keys not read: %+v %v %v", got.JetBrains, got.Base.SwapSizeMB, got.Security.Outbound)
	}
	if got.Base.Locale != "en_US.UTF-8" || !got.JetBrains.Prune {
		t.Error("a key left out lost its default")
	}
}

func TestParseDerivesFromTheUser(t *testing.T) {
	tests := []struct{ in, home, key, gopls string }{
		{"", "/root", "/root/.ssh/id_ed25519_signing", "/root/go/bin/gopls"},
		{"ws_user: dev\n", "/home/dev", "/home/dev/.ssh/id_ed25519_signing", "/home/dev/go/bin/gopls"},
		{"ws_home: /srv/x\n", "/srv/x", "/srv/x/.ssh/id_ed25519_signing", "/srv/x/go/bin/gopls"},
		{"helix_gopls: /bin/g\n", "/root", "/root/.ssh/id_ed25519_signing", "/bin/g"},
	}
	for _, tt := range tests {
		got, err := Parse([]byte(tt.in))
		if err != nil {
			t.Fatal(err)
		}
		if got.Home != tt.home || got.GitSigningKey != tt.key || got.Helix.Gopls != tt.gopls {
			t.Errorf("Parse(%q) = %q %q %q", tt.in, got.Home, got.GitSigningKey, got.Helix.Gopls)
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
