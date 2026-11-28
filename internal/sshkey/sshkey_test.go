package sshkey

import "testing"

func TestCheck(t *testing.T) {
	for name, ok := range map[string]bool{
		"ws-sign": true, "ws-auth": true, "id_ed25519": true, "a": true, "0": true, "A.b_c-d": true, "a..b": true,
		"": false, "-a": false, ".a": false, "_a": false, "../a": false, "a/b": false,
		"../../etc/ssh/ssh_host_ed25519_key": false, "a b": false, "a\n": false, "é": false,
		"a.pub": false, "a.sock": false, "seward.sock": false, "seward.lock": false, "a.lock": false,
		"a.pub.b": true, "pub": true, "sock": true,
	} {
		if err := Check(name); (err == nil) != ok {
			t.Errorf("Check(%q) = %v, want ok %v", name, err, ok)
		}
	}
}
