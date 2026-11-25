package git_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/ubuntu/roles/git"
)

const (
	stripComments = "git stripspace --strip-comments"
	cut           = "; ------------------------ >8 ------------------------\n"
)

// addTrailer is the command that adds the trailer to the message's file, msg.
func addTrailer() string {
	return "git interpret-trailers --in-place --no-divider --if-exists addIfDifferent --trailer " + trailer() + " msg"
}

// commitMsg runs CommitMsg on a message whose text less its comments is stripped. It returns the
// commands that ran, and what the first read on its standard input.
func commitMsg(t *testing.T, text, stripped string, noEditor bool) (ran []string, input string) {
	m := effecttest.NewMachine()
	m.Files["msg"] = []byte(text)
	m.Results[stripComments] = effecttest.Result{Out: []byte(stripped)}
	m.Results[addTrailer()] = effecttest.Result{}
	if err := git.CommitMsg(t.Context(), m, "msg", noEditor); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.Calls {
		if line, ok := strings.CutPrefix(c, "run "); ok {
			ran = append(ran, line)
		}
	}
	return ran, strings.Join(m.Inputs, "")
}

func TestCommitMsgAddsTheMacsTrailer(t *testing.T) {
	// With no editor a line that starts as a comment is text, and git strips none.
	if ran, _ := commitMsg(t, "#42 fix the fan curve\n", "", true); !slices.Equal(ran, []string{addTrailer()}) {
		t.Errorf("with no editor ran %q, want the trailer's command alone", ran)
	}
	ran, input := commitMsg(t, "fix: the fan curve\n# a comment\n"+cut+"diff --git a/x b/x\n", "fix: the fan curve\n", false)
	if !slices.Equal(ran, []string{stripComments, addTrailer()}) {
		t.Errorf("with an editor ran %q, want the strip and then the trailer's command", ran)
	}
	if want := "fix: the fan curve\n# a comment\n"; input != want {
		t.Errorf("git stripspace read %q, want the text above the scissors line, %q", input, want)
	}
}

func TestCommitMsgLeavesAnEmptyMessageAlone(t *testing.T) {
	for _, text := range []string{"", " \n\t\r\n"} {
		if ran, _ := commitMsg(t, text, "", true); len(ran) != 0 {
			t.Errorf("with no editor, %q ran %q, want no command", text, ran)
		}
	}
	for _, text := range []string{"# a comment\n", cut + "diff --git a/x b/x\n"} {
		if ran, _ := commitMsg(t, text, "", false); !slices.Equal(ran, []string{stripComments}) {
			t.Errorf("with an editor, %q ran %q, want the strip alone", text, ran)
		}
	}
}

func TestCommitMsgFailsOnAMessageItCannotRead(t *testing.T) {
	if err := git.CommitMsg(t.Context(), effecttest.NewMachine(), "msg", true); err == nil {
		t.Error("a missing file passed")
	}
}
