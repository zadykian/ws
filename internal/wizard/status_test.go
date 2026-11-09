package wizard

import "testing"

func TestParseStatus(t *testing.T) {
	text := "run v1.2.3 0 off aa11 bb22\nlast ws-wizard-1 0\nactive -\nws v1.2.3\nuser -\nmac.pub mk\n" +
		"wg0 wk\nsigning ssh-ed25519 AAAA\nhostname ws\nclaude login\nfans refused\ncable enp1s0\n" +
		"peer ok\noutbound off\nsecrets aa11\nvars -\nactive ws-wizard-2\n"
	want := wsStatus{Run: stamp{Version: "v1.2.3", Exit: "0", Off: true, Secrets: "aa11", Vars: "bb22"},
		Version: "v1.2.3", MacKey: "mk", WgKey: "wk", Signing: "ssh-ed25519 AAAA", Hostname: "ws",
		Claude: "login", FansRefused: true, Cable: "enp1s0", Peered: true, OutboundOff: true, Secrets: "aa11"}
	if got := parseStatus(text); got != want {
		t.Errorf("parseStatus = %+v; want %+v", got, want)
	}
}

func TestParseStamp(t *testing.T) {
	for text, want := range map[string]stamp{
		"-":                   {},
		"v1 0 on - -":         {Version: "v1", Exit: "0"},
		"- 1 off aa bb":       {Exit: "1", Off: true, Secrets: "aa", Vars: "bb"},
		"v1 0 on - - surplus": {},
	} {
		if got := parseStamp(text); got != want {
			t.Errorf("parseStamp(%q) = %+v; want %+v", text, got, want)
		}
	}
}
