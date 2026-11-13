package setup_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
	"github.com/zadykian/ws/roles/client/macos/setup"
)

const usersToken = "FAKETOKEN0123"

// usersText opens on a secret, as the lines beside ws's at a file's top are what a diff quotes.
const usersText = "[url \"https://oauth2:" + usersToken + "@example.test/\"]\n    insteadOf = https://example.test/\n"

// Each file of the user's that a plan adds to: a report and a log show ws's line. They leave out
// the lines beside it that --check's diff shows.
func TestAReportShowsWsLinesAloneOfAUsersFile(t *testing.T) {
	gitMachine, _, git := gitWithConfig(usersText, "", "")
	gitMachine.Results["git config --file "+gitConfigFile+" --type path --get-all include.path"] =
		effecttest.Result{Err: mactest.ExitOne(t)}
	sshConfig := sshMachine()
	sshConfig.Files["/h/.ssh/config"] = []byte(usersText)
	ghosttyConfig := effecttest.NewMachine()
	ghostty := ghosttyStep(ghosttyConfig, &effecttest.Out{})
	ghosttyConfig.Files["/h/.config/ghostty/config"] = []byte(usersText)
	for _, tt := range []struct {
		name string
		s    step.Step
		line string
	}{
		{"git", git, "+[include]"},
		{"ssh", sshStep(sshConfig, &effecttest.Out{}), "+Include config.d/ws"},
		{"ghostty", ghostty, "+config-file = ?ws.ghostty"},
	} {
		p, err := tt.s.Plan(context.Background())
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		var shown, diffs strings.Builder
		for _, c := range p.Changes {
			shown.WriteString(step.PendingOf(c).Text + step.ExcerptOf(c))
			diffs.WriteString(c.String())
		}
		if strings.Contains(shown.String(), usersToken) || !strings.Contains(shown.String(), tt.line) {
			t.Errorf("%s: report = %q, want %q and none of the user's text", tt.name, shown.String(), tt.line)
		}
		if !strings.Contains(diffs.String(), usersToken) {
			t.Errorf("%s: --check's diff lost its context: %q", tt.name, diffs.String())
		}
	}
}

// ws replaces its own files whole, so what one held stands in the excerpt.
func TestAnExcerptKeepsTheDiffOfAFileWsReplaces(t *testing.T) {
	w := setup.Write{Path: "/f", Before: []byte("old\n"), After: []byte("new\n"), Exists: true}
	if got := step.ExcerptOf(w); got != w.String() {
		t.Errorf("excerpt = %q, want the diff %q", got, w.String())
	}
	w = setup.Write{Path: "/f", After: []byte("new\n"), Exists: true}
	if got := step.ExcerptOf(w); got != w.String() {
		t.Errorf("empty file: excerpt = %q, want the diff %q", got, w.String())
	}
}
