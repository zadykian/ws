package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
)

// wordsStage names the stage in its messages.
const wordsStage = "Word list"

// Words is the stage that copies ws's word list, root's ~/.config/ws/forbidden-words there, which
// the hooks of ws's clone and of another repository's read (docs/checks.md, The hooks), over the
// Mac's where they differ, with or without WS_HOOK. It runs after Hook on ws, whose apply.sh may
// write the list.
type Words struct {
	Host *HostKey
	// Root is the stage that logs in as root: the list is read only where it did, in this run.
	Root *RootKey
	// Hook is Hook on ws, which Plan waits on under --check while WS_HOOK's apply.sh has not run.
	Hook *HookWs
	// Files runs the files script's operations as root on ws.
	Files *Remote
	Home  string
	M     effect.Machine
	Out   effect.Out
}

// Name is the stage's name.
func (*Words) Name() string { return wordsStage }

// WaitsOn is Hook on ws.
func (*Words) WaitsOn() string { return hookWsStage }

// wordsCopy is the change that copies ws's list over the Mac's.
type wordsCopy struct{ file, hash string }

func (c wordsCopy) String() string {
	return wordsStage + ": would copy ws's word list to " + c.file + ", mode 600"
}

func (w *Words) file() string { return w.Home + "/.config/ws/forbidden-words" }

// sums returns git's hash of ws's word list, as the files script prints it, and of the Mac's, each
// "-" where there is none. A Mac's list that the user can't read counts as none: Apply replaces it,
// or stops at a link. git hashes from /, as in a repository it takes that repository's hash, which
// may be SHA-256.
func (w *Words) sums(ctx context.Context) (remote, mac string, err error) {
	out, err := w.Files.Do(ctx, "words")
	remote = strings.TrimSpace(string(out))
	if err != nil || remote == "" {
		return "", "", fmt.Errorf("%s: the files script read no word list on ws: %v", wordsStage, err)
	}
	mac = "-"
	if info, err := w.M.Stat(w.file()); err == nil && info.Mode.IsRegular() {
		if data, err := w.M.ReadFile(w.file()); err == nil {
			sum, err := w.M.RunInput(ctx, data, "git", "-C", "/", "hash-object", "--stdin")
			if err != nil {
				return "", "", fmt.Errorf("%s: git can't hash the Mac's word list, %s, as above", wordsStage, w.file())
			}
			mac = strings.TrimSpace(string(sum))
		}
	}
	return remote, mac, nil
}
