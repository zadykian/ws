package wizard

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/mac"
)

// caseWords returns the words of the case table in the script's main.
func caseWords(script []byte) []string {
	_, body, _ := strings.Cut(string(script), "\nmain() {")
	var words []string
	for _, m := range regexp.MustCompile(`(?m)^    ([a-z]+)\) `).FindAllStringSubmatch(body, -1) {
		words = append(words, m[1])
	}
	slices.Sort(words)
	return words
}

// tableWords returns the words that the table sends to the files script, or to the remote script.
func tableWords(files bool) []string {
	var words []string
	for _, rule := range opRules {
		for _, snd := range rule.sends {
			if snd.files == files && !slices.Contains(words, snd.word) {
				words = append(words, snd.word)
			}
		}
	}
	slices.Sort(words)
	return words
}

func TestOpsMatchTheScripts(t *testing.T) {
	for name, tc := range map[string]struct {
		script []byte
		files  bool
	}{"remote": {mac.RemoteScript, false}, "files": {mac.RemoteFilesScript, true}} {
		if got, want := tableWords(tc.files), caseWords(tc.script); !slices.Equal(got, want) {
			t.Errorf("%s: the table sends %q; the script's case table holds %q", name, got, want)
		}
	}
}

// Each operation leaves the status stale where it changes ws, and else not. Each case reads the
// status, runs the operation, reads the status again, and counts the status ops sent.
func TestEachOperationsStaleRule(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		run     func(*Session) error
		changes bool
	}{
		"ports":     {func(s *Session) error { _, err := s.ports(ctx); return err }, false},
		"waitPorts": {func(s *Session) error { _, err := s.waitPorts(ctx, "eth1"); return err }, false},
		"nameCable": {func(s *Session) error { return s.nameCable(ctx, "eth1") }, true},
		"nameUser":  {func(s *Session) error { return s.nameUser(ctx, "ada") }, true},
		"namePeer":  {func(s *Session) error { return s.namePeer(ctx, "key") }, true},
		"install":   {func(s *Session) error { return s.install(ctx, "v1", "url") }, true},
		"run":       {func(s *Session) error { return s.run(ctx, nil) }, true},
		"outbound":  {func(s *Session) error { return s.outbound(ctx) }, true},
		"values":    {func(s *Session) error { _, err := s.values(ctx); return err }, false},
		"setValues": {func(s *Session) error { return s.setValues(ctx, []string{"a=b"}) }, true},
	} {
		m := effecttest.NewMachine()
		s := macSession(m, SSH{M: m, Lock: "/l"}, &effecttest.Out{}, wsStatus{})
		for _, line := range []string{"link", "link eth1", "wait eth1", "user ada", "peer key", "install v1 url",
			"run", "run --only network,wireguard", "run --only wireguard", "outbound", "values", "values YT1i"} {
			m.Results["op "+line] = effecttest.Result{}
		}
		s.status(ctx)
		if err := tc.run(s); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		s.status(ctx)
		want := 1
		if tc.changes {
			want = 2
		}
		if got := strings.Count(strings.Join(m.Calls, "\n"), "op status"); got != want {
			t.Errorf("%s: the status was read %d times; want %d", name, got, want)
		}
	}
}
