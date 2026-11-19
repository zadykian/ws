package ssh

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// userConfig is ~/.ssh/config as the facts' check reads it.
type userConfig struct {
	Exists bool
	Text   string
	// IncludeLine is the line of its Include of /etc/ws/ssh_config, or 0.
	IncludeLine int
}

// validate stops where ssh refuses the new /etc/ws/ssh_config, alone or as ssh will read it after the
// run. It checks the new file first, then ~/.ssh/config, then the system's file. ~/.ssh/config's
// Include of /etc/ws/ssh_config is left out, as it would read the file that the run replaces. It writes
// its files in a directory of its own, and removes it. ssh -G asks for no terminal by -T: with no
// command and no terminal on stdin, ssh prints that it gives none.
func validate(ctx context.Context, m effect.Machine, home string, u userConfig, hosts []sshcfg.Host, text []byte) error {
	out, err := m.Run(ctx, "mktemp", "-d")
	if err != nil {
		return fmt.Errorf("cannot make a scratch directory: %w", err)
	}
	dir := strings.TrimSpace(string(out))
	defer func() { _, _ = m.Run(ctx, "rm", "-rf", dir) }()
	next := dir + "/new"
	if err := m.WriteFile(next, text, 0o600); err != nil {
		return fmt.Errorf("cannot write %s: %w", next, err)
	}
	for _, h := range hosts {
		out, err := m.Run(ctx, "ssh", "-T", "-G", "-F", next, h.Name)
		if err != nil {
			return fmt.Errorf("ssh refuses the config it would write, as %s", stderrOf(err))
		}
		if err := readsAs(h, out); err != nil {
			return err
		}
	}
	whole := "Include \"" + next + "\"\n"
	user := dir + "/user"
	if u.Exists {
		lines := strings.Split(u.Text, "\n")
		if n := u.IncludeLine; n > 0 && n <= len(lines) {
			lines[n-1] = ""
		}
		if err := m.WriteFile(user, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
			return fmt.Errorf("cannot write %s: %w", user, err)
		}
		whole += "Match all\nInclude \"" + user + "\"\n"
	}
	if _, err := m.Stat("/etc/ssh/ssh_config"); !errors.Is(err, fs.ErrNotExist) {
		whole += "Match all\nInclude /etc/ssh/ssh_config\n"
	}
	all := dir + "/whole"
	if err := m.WriteFile(all, []byte(whole), 0o600); err != nil {
		return fmt.Errorf("cannot write %s: %w", all, err)
	}
	for _, h := range hosts {
		out, err := m.Run(ctx, "ssh", "-T", "-G", "-F", all, h.Name)
		if err == nil {
			if err := readsAs(h, out); err != nil {
				return err
			}
			continue
		}
		e := strings.ReplaceAll(stderrOf(err), user, home+"/.ssh/config")
		switch {
		case strings.Contains(e, sshcfg.File):
			return fmt.Errorf("ssh refuses /etc/ws/ssh_config, which ~/.ssh/config also reads, as %s: "+
				"remove that Include, and run mise -C /opt/ws/roles/client bootstrap again", e)
		case strings.Contains(e, "Bad configuration option"):
			return fmt.Errorf("ssh refuses ~/.ssh/config read after the new /etc/ws/ssh_config, as %s; "+
				"its IgnoreUnknown, read first, carries only ~/.ssh/config's own "+
				"patterns, so one for a file it includes must be there too", e)
		}
		return fmt.Errorf("ssh refuses ~/.ssh/config read after the new /etc/ws/ssh_config, as %s", e)
	}
	return nil
}

// readsAs checks the output of ssh -G for h against h's block. A hostname or a user that ssh
// prints must be the block's.
func readsAs(h sshcfg.Host, out []byte) error {
	for _, l := range strings.Split(string(out), "\n") {
		key, val, _ := strings.Cut(l, " ")
		switch {
		case key == "hostname" && val != h.Addr.String(), key == "user" && val != h.User:
			return fmt.Errorf("ssh reads %s as %s: /etc/ws/ssh_config sets %s at %s, which another file "+
				"of ssh's changes", h.Name, strings.TrimSpace(l), h.User, h.Addr)
		}
	}
	return nil
}

// stderrOf is the text of a failed command's error, which holds its standard error.
func stderrOf(err error) string {
	_, msg, _ := strings.Cut(err.Error(), ": exit status")
	_, msg, found := strings.Cut(msg, ": ")
	if !found {
		return err.Error()
	}
	return msg
}
