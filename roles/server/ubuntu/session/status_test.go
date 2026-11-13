package session

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestParseStatus(t *testing.T) {
	text := "run v1.2.3 0 off aa11 bb22\nlast ws-wizard-1 0\nactive -\nws v1.2.3\nuser -\nmac.pub mk\n" +
		"wg0 wk\nsigning ssh-ed25519 AAAA\nhostname ws\nclaude login\nfans refused\ncable enp1s0\n" +
		"peer ok\noutbound off\nsecrets aa11\nvars -\nactive ws-wizard-2\n"
	want := Status{Run: Stamp{Version: "v1.2.3", Exit: "0", Off: true, Secrets: "aa11", Vars: "bb22"},
		Version: "v1.2.3", MacKey: "mk", WgKey: "wk", Signing: "ssh-ed25519 AAAA", Hostname: "ws",
		Claude: "login", FansRefused: true, Cable: "enp1s0", Peered: true, OutboundOff: true, Secrets: "aa11"}
	if got := parseStatus(text); got != want {
		t.Errorf("parseStatus = %+v; want %+v", got, want)
	}
}

func TestParseStamp(t *testing.T) {
	for text, want := range map[string]Stamp{
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

// A line of each name that the remote script's status prints, with a value that sets its field.
// last is the one the wizard leaves.
var statusLines = map[string]string{
	"run": "v1 0 on - -", "active": "u", "ws": "v1", "user": "u", "mac.pub": "k", "wg0": "k",
	"signing": "none", "hostname": "h", "claude": "ok", "fans": "refused", "cable": "p", "peer": "ok",
	"outbound": "off", "secrets": "s", "vars": "v",
}

// The parse reads each name that the script's status prints, and the stamp's words in the order
// that the script's job writes them.
func TestStatusPinsScript(t *testing.T) {
	script := string(RemoteScript)
	start := strings.Index(script, "\nstatus() {\n")
	end := strings.Index(script[start+1:], "\n}\n")
	if start < 0 || end < 0 {
		t.Fatal("the script has no status function")
	}
	var names []string
	for _, m := range regexp.MustCompile(`echo ["']([a-z][a-z0-9.]*) `).FindAllStringSubmatch(script[start:start+end], -1) {
		names = append(names, m[1])
	}
	want := []string{"last"}
	for name := range statusLines {
		want = append(want, name)
	}
	slices.Sort(names)
	names = slices.Compact(names)
	slices.Sort(want)
	if !slices.Equal(names, want) {
		t.Errorf("the script's status prints %v; the parse reads %v", names, want)
	}
	for name, value := range statusLines {
		if parseStatus(name+" "+value) == (Status{}) {
			t.Errorf("the parse leaves %s", name)
		}
	}
	if !strings.Contains(script, `echo "$version $status $table $sum $values" >"$state/run.new"`) {
		t.Error("the script's job writes a stamp of other words")
	}
}
