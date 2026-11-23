package git

import (
	"bytes"
	"context"
	"regexp"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/internal/gitcfg"
)

// scissors starts the section that git cuts from an edited message: its comment string, a space,
// 24 dashes, >8 and 24 dashes.
var scissors = regexp.MustCompile(`(?m)^\S* -{24} >8 -{24}$`)

// CommitMsg adds the maintainer's Co-authored-by trailer to the message in file, once. An empty
// message gets none, so that git aborts the commit. noEditor is git's GIT_EDITOR=:, under which a
// comment line is text (README.md#the-co-author-trailer).
func CommitMsg(ctx context.Context, m effect.Machine, file string, noEditor bool) error {
	text, err := m.ReadFile(file)
	if err != nil {
		return err
	}
	if !noEditor {
		if at := scissors.FindIndex(text); at != nil {
			text = text[:at[0]]
		}
		if text, err = m.RunInput(ctx, text, "git", "stripspace", "--strip-comments"); err != nil {
			return err
		}
	}
	// The set is git's own whitespace, by which it tells an empty message.
	if len(bytes.Trim(text, " \t\r\n")) == 0 {
		return nil
	}
	mac := gitcfg.Mac()
	// addIfDifferent adds no second trailer, and --no-divider keeps it below a line of three dashes.
	_, err = m.Run(ctx, "git", "interpret-trailers", "--in-place", "--no-divider", "--if-exists", "addIfDifferent",
		"--trailer", "Co-authored-by: "+mac.Name+" <"+mac.Email+">", file)
	return err
}
