package kind

import (
	"cmp"
	"errors"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/file"
)

// fileByHand is a file's fields for a file that its owner edits by hand too, as a user's own
// config is.
type fileByHand struct {
	// LineAt puts Line at the file's "top", the default, or at its "end", where a format reads its
	// last setting last.
	LineAt string `toml:"line_at"`
	// Mark is the comment sign before ws's mark, for ours: "#" where unset, "//" for JSON with
	// comments.
	Mark string `toml:"mark"`
	// KeepForeign leaves a file that lacks the mark, with a warning, where ours would fail the plan:
	// the user may keep a file of their own there.
	KeepForeign bool `toml:"keep_foreign"`
}

// checkByHand refuses the fields of fileByHand that no plan could take.
func (f filePart) checkByHand() error {
	switch {
	case f.LineAt != "" && f.Line == "":
		return errors.New("line_at goes with line")
	case f.LineAt != "" && f.LineAt != "top" && f.LineAt != "end":
		return fmt.Errorf("line_at %q is top or end", f.LineAt)
	case (f.Mark != "" || f.KeepForeign) && !f.Ours:
		return errors.New("mark and keep_foreign go with ours")
	case f.Mark != "" && strings.ContainsAny(f.Mark, " \t\r\n"):
		return fmt.Errorf("mark %q is a comment sign, with no white space", f.Mark)
	}
	return nil
}

// guarded returns the file's resource behind its guards: ours, which reads the file's first line
// before a write or a removal.
func (f filePart) guarded(e Env, owner account.Account, r resource.Resource) resource.Resource {
	if !f.Ours {
		return r
	}
	return ours{m: e.M, out: e.Out, path: f.Path, mark: cmp.Or(f.Mark, "#") + " " + file.Mark,
		home: owner.User, keep: f.KeepForeign, file: r}
}
