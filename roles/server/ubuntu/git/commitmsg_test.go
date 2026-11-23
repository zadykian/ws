package git_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/git"
)

// addTrailer is the command that adds the trailer to the message's file, msg.
func addTrailer() string {
	return "git interpret-trailers --in-place --no-divider --if-exists addIfDifferent --trailer " + trailer() + " msg"
}

// commitMsg runs CommitMsg on a message, and returns the commands that ran.
func commitMsg(t *testing.T, text string) (ran []string) {
	m := effecttest.NewMachine()
	m.Files["msg"] = []byte(text)
	m.Results[addTrailer()] = effecttest.Result{}
	if err := git.CommitMsg(t.Context(), m, "msg"); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.Calls {
		if line, ok := strings.CutPrefix(c, "run "); ok {
			ran = append(ran, line)
		}
	}
	return ran
}

func TestCommitMsgAddsTheMacsTrailer(t *testing.T) {
	if ran := commitMsg(t, "#42 fix the fan curve\n"); !slices.Equal(ran, []string{addTrailer()}) {
		t.Errorf("ran %q, want the trailer's command alone", ran)
	}
}

func TestCommitMsgLeavesAnEmptyMessageAlone(t *testing.T) {
	for _, text := range []string{"", " \n\t\r\n"} {
		if ran := commitMsg(t, text); len(ran) != 0 {
			t.Errorf("%q ran %q, want no command", text, ran)
		}
	}
}

func TestCommitMsgFailsOnAMessageItCannotRead(t *testing.T) {
	if err := git.CommitMsg(t.Context(), effecttest.NewMachine(), "msg"); err == nil {
		t.Error("a missing file passed")
	}
}
