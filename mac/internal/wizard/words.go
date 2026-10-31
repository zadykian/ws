package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
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
	if err != nil {
		return "", "", fmt.Errorf("%s: the files script read no word list on ws: %w", wordsStage, err)
	}
	if remote == "" {
		return "", "", fmt.Errorf("%s: the files script read no word list on ws", wordsStage)
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

// Plan is empty where ws has no word list, or the Mac's is ws's. It reads ws only where root logged
// in there in this run. Where the Mac's is a link and ws's list differs, it stops, under --check too,
// naming the link's target, as Apply would replace the link.
func (w *Words) Plan(ctx context.Context) (step.Plan, error) {
	way, err := w.Host.SSH.Way(ctx)
	if err != nil {
		return step.Plan{}, fmt.Errorf("%s: %w", wordsStage, err)
	}
	if ok, _ := w.Root.login(ctx, way); !ok {
		w.Out.Say("%s: after %s", wordsStage, rootStage)
		return step.Plan{}, nil
	}
	if c, err := w.Host.conf(); err == nil && c.Hook != "" && w.Host.Check && !w.hookDone(ctx, c.Hook) {
		w.Out.Say("%s: after %s", wordsStage, hookWsStage)
		return step.Plan{}, nil
	}
	remote, mac, err := w.sums(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	if remote == "-" || remote == mac {
		return step.Plan{}, nil
	}
	if info, _ := w.M.Stat(w.file()); info.Link != "" {
		return step.Plan{}, fmt.Errorf("%s: %s is a link to %s, which the wizard leaves, and ws's word list differs: "+
			"remove the link, or copy root's ~/.config/ws/forbidden-words on ws to that file, then run the "+
			"wizard again", wordsStage, w.file(), info.Link)
	}
	return step.Plan{Changes: []step.Change{wordsCopy{w.file(), remote}}}, nil
}

// hookDone reports whether ws has the stamp of the hook's apply.sh.
func (w *Words) hookDone(ctx context.Context, hook string) bool {
	return w.Hook.Run.Remote.Status(ctx) && strings.Contains(" "+w.Hook.Run.Remote.Get("hook")+" ", " "+hook+" ")
}

// Apply copies ws's list over the Mac's, mode 600. The list goes through memory alone, is never
// printed, and is written only if its hash is the one read before.
func (w *Words) Apply(ctx context.Context, p step.Plan) error {
	for _, ch := range p.Changes {
		c, ok := ch.(wordsCopy)
		if !ok {
			continue
		}
		data, err := w.Files.Do(ctx, "words", c.hash)
		sum := []byte(nil)
		if err == nil {
			sum, err = w.M.RunInput(ctx, data, "git", "-C", "/", "hash-object", "--stdin")
		}
		if err != nil || strings.TrimSpace(string(sum)) != c.hash {
			return fmt.Errorf("%s: ws's word list came short or changed, as above; run the wizard again", wordsStage)
		}
		for _, d := range []string{w.Home + "/.config", w.Home + "/.config/ws"} {
			_ = w.M.Mkdir(d, 0o700) // exists mostly; WriteFile fails where the directory is missing
		}
		// A new file renamed over the old one, as ws_put did, so that a list the Mac's user can't
		// write, such as a mode 400 one, is replaced all the same.
		tmp := c.file + ".new"
		if err := w.M.WriteFile(tmp, data, 0o600); err != nil {
			return err
		}
		if _, err := w.M.Run(ctx, "mv", "-f", tmp, c.file); err != nil {
			_, _ = w.M.Run(ctx, "rm", "-f", tmp)
			return err
		}
	}
	return nil
}
