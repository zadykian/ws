package conf

import "testing"

func TestScopeRefusesTheOtherFilesKeys(t *testing.T) {
	for _, c := range []struct {
		scope Scope
		text  string
		want  string
	}{
		{Both, "WS_TARGET=lan\nWS_GIT_NAME=Ada\n", ""},
		{MacFile, "WS_GIT_NAME=Ada\n", ""},
		{ServerFile, "WS_TARGET=lan\n", ""},
		{MacFile, "WS_TARGET=lan\n", "WS_TARGET is a server's key: it goes in servers/NAME/server.conf, beside mac.conf"},
		{ServerFile, "WS_GIT_NAME=Ada\n", "WS_GIT_NAME is the Mac's key: it goes in mac.conf"},
	} {
		_, errs := c.scope.Lines(c.text)
		if c.want == "" {
			if len(errs) != 0 {
				t.Errorf("scope %d, %q: errors %v, want none", c.scope, c.text, errs)
			}
			continue
		}
		if len(errs) != 1 || errs[0] != (Error{1, c.want}) {
			t.Errorf("scope %d, %q: errors %v, want line 1: %s", c.scope, c.text, errs, c.want)
		}
	}
}

func TestScopeParseKeepsTheKeysThatGoTogether(t *testing.T) {
	_, errs := ServerFile.Parse("WS_LAN=10.88.0.0/24\n")
	if len(errs) != 1 || errs[0].Msg != "WS_LAN and WS_LAN_ADDRESS go together" {
		t.Errorf("errors = %v, want the pair's refusal", errs)
	}
	if _, errs := MacFile.Parse("WS_GIT_NAME=Ada\nWS_GIT_EMAIL=a@b.co\n"); len(errs) != 0 {
		t.Errorf("errors = %v, want none", errs)
	}
}

func TestEveryKeyHasAScope(t *testing.T) {
	for _, k := range macKeys {
		if _, ok := lookup(k); !ok {
			t.Errorf("%s is listed as the Mac's but is no key", k)
		}
	}
}
