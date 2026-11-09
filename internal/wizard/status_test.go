package wizard

import (
	"cmp"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac"
)

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

// text is the status as the remote script prints it, with - for an empty field, so that a stage's
// test states ws's status as a value.
func (s wsStatus) text() []byte {
	word := func(set bool, yes, no string) string {
		if set {
			return yes
		}
		return no
	}
	run := "-"
	if s.Run != (stamp{}) {
		run = strings.Join([]string{cmp.Or(s.Run.Version, "-"), cmp.Or(s.Run.Exit, "-"), word(s.Run.Off, "off", "on"),
			cmp.Or(s.Run.Secrets, "-"), cmp.Or(s.Run.Vars, "-")}, " ")
	}
	lines := []string{"run " + run, "fans " + word(s.FansRefused, "refused", "ok"), "peer " + word(s.Peered, "ok", "-"),
		"outbound " + word(s.OutboundOff, "off", "on")}
	for name, value := range map[string]string{"active": s.Active, "ws": s.Version, "user": s.User,
		"mac.pub": s.MacKey, "wg0": s.WgKey, "signing": s.Signing, "hostname": s.Hostname, "claude": s.Claude,
		"cable": s.Cable, "secrets": s.Secrets, "vars": s.Vars} {
		lines = append(lines, name+" "+cmp.Or(value, "-"))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

// The text that a test states parses back to its value.
func TestStatusText(t *testing.T) {
	full := parseStatus("run v1 0 off aa bb\nactive u\nws v1\nuser alice\nmac.pub mk\nwg0 wk\n" +
		"signing none\nhostname ws\nclaude ok\nfans refused\ncable p\npeer ok\noutbound off\nsecrets s\nvars v\n")
	for _, s := range []wsStatus{{}, full} {
		if got := parseStatus(string(s.text())); got != s {
			t.Errorf("parseStatus(%q) = %+v; want %+v", s.text(), got, s)
		}
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
	script := string(mac.RemoteScript)
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
		if parseStatus(name+" "+value) == (wsStatus{}) {
			t.Errorf("the parse leaves %s", name)
		}
	}
	if !strings.Contains(script, `echo "$version $status $table $sum $values" >"$state/run.new"`) {
		t.Error("the script's job writes a stamp of other words")
	}
}
