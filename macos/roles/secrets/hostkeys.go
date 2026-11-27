package secrets

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/macos/values"
)

// HostKeys changes the host keys that the secrets repository holds, as ws macos secrets add and
// remove do. Each change runs in a temporary clone of the repository and lands by a pull request.
type HostKeys struct {
	M      effect.Machine
	Getenv func(string) string
	// Out takes what the change says, the pull request's link among it.
	Out io.Writer
	// KeyFile is fnox's age key file, whose stub the Nano's PIN and touch open.
	KeyFile string
}

// edited is a clone whose files an edit changed, and whose values fnox encrypted again.
type edited struct {
	clone   *Clone
	repo    Repo
	keys    []DeployKey
	files   Files
	changed []string
}

// Add makes key the recipient of hostname in the secrets, keeps it as the repository's deploy key
// titled hostname, and lands the change. The deploy keys are read first, so that gh's login fails
// before the Nano asks for a touch. A change of no recipient lands nothing. server is the Mac's name
// for the machine. byHand is a line for each host key that the secrets keep and that known_hosts
// gives no host. A server that came back with another hostname leaves its older key so.
func (h HostKeys) Add(ctx context.Context, server, hostname, key string) (byHand []string, err error) {
	known, err := knownKeys(h.M, h.Getenv("HOME"))
	if err != nil {
		return nil, err
	}
	set := func(f Files) (Files, []string, error) { return setHostKey(f, hostname, key) }
	err = h.change(ctx, set, func(e edited) error {
		// GitHub changes no deploy key in place, so an older key goes before the new one comes.
		drop, add := planDeployKeys(e.keys, hostname, key)
		for _, k := range drop {
			if err := h.deleteKey(ctx, e.repo, k); err != nil {
				return err
			}
		}
		if add {
			if err := addDeployKey(ctx, h.M, GH, e.repo.Slug(), hostname, key); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(h.Out, "added the read-only deploy key %s to %s\n", hostname, e.repo.Slug())
		}
		kept := slices.DeleteFunc(e.keys, func(k DeployKey) bool { return slices.Contains(drop, k) })
		left, err := strays(e.files, kept, known)
		if err != nil {
			return err
		}
		for _, s := range left {
			byHand = append(byHand, fmt.Sprintf("By hand: the secrets hold %s under %s, a key of no host in "+
				"known_hosts: where no server holds it now, as after a rebuild, drop it from the secrets' "+
				"recipients and deploy keys", s.Key, values.List(s.Names)))
		}
		if len(e.changed) == 0 {
			_, _ = fmt.Fprintf(h.Out, "the secrets already hold the host key of %s\n", hostname)
			return nil
		}
		_, err = e.clone.Land(ctx, h.Out, Change{Topic: "add-" + hostname, Title: "chore: add the host key of " + hostname,
			Body: "Added by ws macos secrets add " + server + ".", Files: e.changed})
		return err
	})
	if err != nil {
		return nil, err
	}
	return byHand, nil
}

// Drop takes key out of the recipients in the secrets and lands the change. Then it deletes each
// deploy key that holds key, whatever its title, as a server whose hostname changed may hold it
// under another. server is the Mac's name for the machine.
func (h HostKeys) Drop(ctx context.Context, server, key string) error {
	var labels []string
	drop := func(f Files) (out Files, changed []string, err error) {
		out, changed, labels, err = dropKey(f, key)
		return out, changed, err
	}
	return h.change(ctx, drop, func(e edited) error {
		if len(e.changed) == 0 {
			_, _ = fmt.Fprintf(h.Out, "the secrets' recipients hold no host key of %s\n", server)
		} else if _, err := e.clone.Land(ctx, h.Out, Change{Topic: "remove-" + strings.Join(labels, "-"),
			Title: "chore: remove the host key of " + values.List(labels),
			Body:  "Removed by ws macos remove " + server + ".", Files: e.changed}); err != nil {
			return err
		}
		for _, k := range e.keys {
			if k.Key != key {
				continue
			}
			if err := h.deleteKey(ctx, e.repo, k); err != nil {
				return err
			}
		}
		return nil
	})
}

// change clones the repository, edits its fnox files by edit, writes them and encrypts their values
// again, then runs finish while the clone stays.
func (h HostKeys) change(ctx context.Context, edit func(Files) (Files, []string, error), finish func(edited) error) error {
	repo, err := RepoOf(h.M, Tree)
	if err != nil {
		return err
	}
	clone, err := Open(ctx, h.M, repo)
	if err != nil {
		return err
	}
	defer clone.Close(ctx)
	clone.FnoxEnv = fnoxEnv(h.Getenv, h.KeyFile)
	keys, err := deployKeys(ctx, h.M, GH, repo.Slug())
	if err != nil {
		return err
	}
	files, err := readFiles(h.M, clone.Dir)
	if err != nil {
		return err
	}
	files, changed, err := edit(files)
	if err != nil {
		return err
	}
	if err := writeFiles(h.M, clone.Dir, files, changed); err != nil {
		return err
	}
	if err := reencrypt(ctx, clone, files, changed); err != nil {
		return err
	}
	return finish(edited{clone: clone, repo: repo, keys: keys, files: files, changed: changed})
}

// deleteKey deletes the deploy key k of the repository and says so.
func (h HostKeys) deleteKey(ctx context.Context, repo Repo, k DeployKey) error {
	if err := deleteDeployKey(ctx, h.M, GH, repo.Slug(), k.ID); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(h.Out, "deleted the deploy key %s of %s\n", k.Title, repo.Slug())
	return nil
}
