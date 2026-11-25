package values

import (
	"strings"
	"testing"
)

func TestSetReplacesTheKeysLineAndAddsTheOthers(t *testing.T) {
	for name, tt := range map[string]struct{ text, want string }{
		"an empty file": {"", "WS_IDENTITY_PUB_KEY: /k.pub\nWS_IDENTITY_AGENT: ~/s\n"},
		"an empty value": {"WS_IDENTITY_PUB_KEY: \"\"\nWS_SLOT: 0\n",
			"WS_IDENTITY_PUB_KEY: /k.pub\nWS_SLOT: 0\nWS_IDENTITY_AGENT: ~/s\n"},
		"comments stay": {"# the Mac\nWS_SLOT: 0 # one\n\nWS_IDENTITY_AGENT: ~/old\n",
			"# the Mac\nWS_SLOT: 0 # one\n\nWS_IDENTITY_AGENT: ~/s\nWS_IDENTITY_PUB_KEY: /k.pub\n"},
		"no last newline": {"WS_SLOT: 0", "WS_SLOT: 0\nWS_IDENTITY_PUB_KEY: /k.pub\nWS_IDENTITY_AGENT: ~/s\n"},
		"a longer key":    {"WS_IDENTITY_PUB_KEY_X: 1\n", "WS_IDENTITY_PUB_KEY_X: 1\nWS_IDENTITY_PUB_KEY: /k.pub\nWS_IDENTITY_AGENT: ~/s\n"},
	} {
		t.Run(name, func(t *testing.T) {
			got := Set(tt.text, "WS_IDENTITY_PUB_KEY", "/k.pub", "WS_IDENTITY_AGENT", "~/s")
			if got != tt.want {
				t.Errorf("Set = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetWritesWhatTheReaderTakesBack(t *testing.T) {
	text := Set("WS_SLOT: 1\n", "WS_IDENTITY_PUB_KEY", "/etc/ws/secretive/ab.pub", "WS_GIT_SIGNING_PUB_KEY", "")
	c, errs := Parse(text)
	if len(errs) > 0 || c.IdentityPubKey != "/etc/ws/secretive/ab.pub" || c.Slot != "1" {
		t.Errorf("Parse(%q) = %+v, %v", text, c, errs)
	}
}

func TestSetPanicsOnAKeyWithNoValue(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Set(text, KEY) did not panic")
		}
	}()
	Set("", strings.Fields("WS_SLOT")...)
}
