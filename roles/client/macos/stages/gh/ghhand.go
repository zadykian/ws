package gh

import (
	"context"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"strings"
)

// Hand adds gh auth setup-git to what stays by hand, where git's config names for github.com no
// credential command of gh's. That command names one, as ws client macos apply's gitconfig does
// where mac.conf gives git a name. Through it, pushes from the Mac's clones take gh's login.
func (g *Gh) Hand(ctx context.Context, hand *guided.Hand) {
	out, _ := g.M.Run(ctx, "git", "config", "--global", "--includes", "--get-all",
		"credential.https://github.com.helper")
	for _, l := range strings.Split(string(out), "\n") {
		if strings.HasSuffix(l, " auth git-credential") {
			return
		}
	}
	hand.Add("Pushes from the Mac's clones to GitHub: gh auth setup-git, which has git take gh's login " +
		"(docs/mac-git.md).")
}
