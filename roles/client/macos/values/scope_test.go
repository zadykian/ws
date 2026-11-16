package values

import "testing"

func TestScopeRefusesTheOtherFilesKeys(t *testing.T) {
	for _, c := range []struct {
		scope Scope
		text  string
		want  string
	}{
		{Both, Text("WS_TARGET", "lan", "WS_GIT_SIGNING_PUB_KEY", "~/k.pub"), ""},
		{MacFile, Text("WS_GIT_SIGNING_PUB_KEY", "~/k.pub"), ""},
		{ServerFile, Text("WS_TARGET", "lan"), ""},
		{MacFile, Text("WS_TARGET", "lan"), "WS_TARGET is a server's key: it goes in servers/NAME/values.yml, beside the Mac's values.yml"},
		{ServerFile, Text("WS_GIT_SIGNING_PUB_KEY", "~/k.pub"), "WS_GIT_SIGNING_PUB_KEY is the Mac's key: it goes in the Mac's values.yml"},
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
	_, errs := ServerFile.Parse(Text("WS_LAN", "10.88.0.0/24"))
	if len(errs) != 1 || errs[0].Msg != "WS_LAN and WS_LAN_ADDRESS go together" {
		t.Errorf("errors = %v, want the pair's refusal", errs)
	}
	if _, errs := MacFile.Parse(Text("WS_GIT_SIGNING_PUB_KEY", "~/k.pub")); len(errs) != 0 {
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

func TestScopeKeepSplitsLinesByKey(t *testing.T) {
	var lines = Text("WS_TARGET", "lan", "WS_GIT_SIGNING_PUB_KEY", "~/k.pub", "WS_REMOTE", "")
	for scope, want := range map[Scope]string{
		MacFile: Text("WS_GIT_SIGNING_PUB_KEY", "~/k.pub"), ServerFile: Text("WS_TARGET", "lan", "WS_REMOTE", ""), Both: lines,
	} {
		if got := scope.Keep(lines); got != want {
			t.Errorf("scope %d keeps %q, want %q", scope, got, want)
		}
	}
}
