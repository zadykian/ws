package host

import (
	"context"
	"errors"
	"fmt"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"io/fs"
	"strings"
)

// Apply adds the key.
func (h *Key) Apply(ctx context.Context, _ step.Plan) error { return h.add(ctx) }

// add adds the ed25519 key that ws offers at its way to known_hosts, under ws, once WS_HOST_KEY, or
// the answer at the terminal, says that ws's console shows it. Where ws doesn't answer at its way,
// it reads the key at ws's address now. It warns where the way is WS_REMOTE. Under --check it asks
// nothing, and shows the line it would add. Where known_hosts is a link, as a dotfiles manager makes
// it, it prints that line, and stops: a write would replace the link. --check, which compared no key
// with ws's console, stops without it.
func (h *Key) add(ctx context.Context) error {
	c, err := h.conf()
	if err != nil {
		return err
	}
	way, err := h.SSH.Way(ctx)
	if err != nil {
		return noWay(h.SSH.Host, c)
	}
	line := scanKey(ctx, h.M, way)
	if line == "" {
		if h.Check {
			h.Out.Say("%s: %s; a run would ask its address now", stage.Host, noAnswer(h.SSH.Host, way))
			return nil
		}
		if way, line, err = h.question().ask(ctx, c.Target, way); err != nil {
			return err
		}
	}
	// ssh -G prints the host name in lower case, which WS_REMOTE may not be.
	if way.Host == strings.ToLower(c.Remote) {
		h.Out.Say("%s: warning: through WS_REMOTE, %s's new sshd faces the internet, with any password "+
			"login its installer allowed, until ws's run sets up its security role", stage.Host, h.SSH.Host)
	}
	fp, err := h.fingerprint(ctx, line)
	if err != nil {
		return err
	}
	if err := h.trust(ctx, way, fp); err != nil {
		return err
	}
	return h.write(way.KnownHosts, line)
}

// write adds the server's line to known_hosts, under its name,, or shows it under --check.
func (h *Key) write(file, line string) error {
	info, err := h.M.Stat(file)
	if err == nil && info.Link != "" {
		why := fmt.Sprintf("%s is a link to %s, which the wizard leaves", file, info.Link)
		if h.Check {
			return fmt.Errorf("%s: %s: a run would print %s's line for its file", stage.Host, why, h.SSH.Host)
		}
		h.Out.Say("%s: %s: add this line to its file, then run the wizard again", stage.Host, why)
		h.Out.Raw("%s %s", h.SSH.Host, line)
		return fmt.Errorf("%s: %s", stage.Host, "known_hosts is left as it is")
	}
	mode := fs.FileMode(0o600)
	if err == nil {
		mode = info.Mode.Perm()
	}
	before, err := h.M.ReadFile(file)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s: cannot read %s", stage.Host, file)
	}
	after := string(before)
	if after != "" && !strings.HasSuffix(after, "\n") {
		after += "\n"
	}
	after += h.SSH.Host + " " + line + "\n"
	if h.Check {
		h.Out.Diff(file, before, []byte(after))
		return nil
	}
	return h.M.WriteFile(file, []byte(after), mode)
}
