package git

import (
	"bytes"
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/internal/gitcfg"
)

// CommitMsg adds the maintainer's Co-authored-by trailer to the message in file, once. An empty
// message gets none, so that git aborts the commit (README.md#the-co-author-trailer).
func CommitMsg(ctx context.Context, m effect.Machine, file string) error {
	text, err := m.ReadFile(file)
	if err != nil {
		return err
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
