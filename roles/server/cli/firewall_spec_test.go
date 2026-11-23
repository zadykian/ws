package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// firewallFile holds mise's firewall.
const firewallFile = rolesDir + "/security/mise.server.toml"

// sshRule is mise's rule that lets SSH in, whole. Once ufw denies what comes in, the maintainer's
// login hangs on it: a source, another port, protocol or action would lock them out.
var sshRule = map[string]any{
	"name": "ssh", "direction": "incoming", "action": "allow", "protocol": "tcp", "port": int64(22),
}

// firewall is the part of a mise file that sets up ufw.
type firewall struct {
	Backend         string           `toml:"backend"`
	DefaultIncoming string           `toml:"default_incoming"`
	DefaultOutgoing string           `toml:"default_outgoing"`
	Rules           []map[string]any `toml:"rules"`
}

// readFirewall returns the firewall of the mise file at path.
func readFirewall(t *testing.T, path string) firewall {
	text, err := os.ReadFile(path) //nolint:gosec // a file of the tree, found by the test
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Bootstrap struct {
			Linux struct {
				Firewall firewall `toml:"firewall"`
			} `toml:"linux"`
		} `toml:"bootstrap"`
	}
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return c.Bootstrap.Linux.Firewall
}

// TestFirewallLetsSSHIn holds what main's ufw step tested: ufw, incoming denied, outgoing allowed,
// and the rule ssh as sshRule gives it. No CI job logs in over SSH, so this test is the guard.
func TestFirewallLetsSSHIn(t *testing.T) {
	fw := readFirewall(t, firewallFile)
	if fw.Backend != "ufw" || fw.DefaultIncoming != "deny" || fw.DefaultOutgoing != "allow" {
		t.Errorf("the firewall is %q, incoming %q, outgoing %q; want ufw, deny, allow",
			fw.Backend, fw.DefaultIncoming, fw.DefaultOutgoing)
	}
	var ssh []map[string]any
	for _, r := range fw.Rules {
		if r["name"] == sshRule["name"] {
			ssh = append(ssh, r)
		}
	}
	if len(ssh) != 1 || !reflect.DeepEqual(ssh[0], sshRule) {
		t.Errorf("the rules named ssh are %v; want %v alone", ssh, sshRule)
	}
}

// TestFirewallSSHRuleIsSecuritysAlone fails where another role's mise file names the rule ssh, or
// a rule on port 22. As mise merges rules by name, such a rule could replace or undo sshRule.
func TestFirewallSSHRuleIsSecuritysAlone(t *testing.T) {
	for _, name := range roleFiles {
		files, err := filepath.Glob(filepath.Join(rolesDir, "*", name))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if f == filepath.Clean(firewallFile) {
				continue
			}
			for _, r := range readFirewall(t, f).Rules {
				if r["name"] == sshRule["name"] || r["port"] == sshRule["port"] {
					t.Errorf("%s: rule %v; the rule on port 22 is security's alone", f, r)
				}
			}
		}
	}
}
