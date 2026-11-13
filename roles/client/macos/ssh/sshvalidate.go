package ssh

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// validate stops where ssh refuses the new config.d/ws, alone or as ssh will read it after the
// run. It checks the new file first, then ~/.ssh/config, then the system's file. ~/.ssh/config's
// Include of config.d/ws is left out, as it would read the file that the run replaces. It writes
// its files in a directory of its own, and removes it.
func (s SSH) validate(ctx context.Context, f sshFacts, hosts []sshcfg.Host, text []byte) error {
	out, err := s.M.Run(ctx, "mktemp", "-d")
	if err != nil {
		return fmt.Errorf("cannot make a scratch directory: %w", err)
	}
	dir := strings.TrimSpace(string(out))
	defer func() { _, _ = s.M.Run(ctx, "rm", "-rf", dir) }()
	next := dir + "/new"
	if err := s.M.WriteFile(next, text, 0o600); err != nil {
		return fmt.Errorf("cannot write %s: %w", next, err)
	}
	for _, h := range hosts {
		out, err := s.M.Run(ctx, "ssh", "-G", "-F", next, h.Name)
		if err != nil {
			return fmt.Errorf("ssh refuses the config it would write, as %s", stderrOf(err))
		}
		if err := readsAs(h, out); err != nil {
			return err
		}
	}
	whole := "Include \"" + next + "\"\n"
	user := dir + "/user"
	if f.UserConfig.Exists {
		lines := strings.Split(string(f.UserConfig.Text), "\n")
		if n := f.IncludeLine; n > 0 && n <= len(lines) {
			lines[n-1] = ""
		}
		if err := s.M.WriteFile(user, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
			return fmt.Errorf("cannot write %s: %w", user, err)
		}
		whole += "Match all\nInclude \"" + user + "\"\n"
	}
	if _, err := s.M.Stat("/etc/ssh/ssh_config"); !errors.Is(err, fs.ErrNotExist) {
		whole += "Match all\nInclude /etc/ssh/ssh_config\n"
	}
	all := dir + "/whole"
	if err := s.M.WriteFile(all, []byte(whole), 0o600); err != nil {
		return fmt.Errorf("cannot write %s: %w", all, err)
	}
	for _, h := range hosts {
		out, err := s.M.Run(ctx, "ssh", "-G", "-F", all, h.Name)
		if err == nil {
			if err := readsAs(h, out); err != nil {
				return err
			}
			continue
		}
		e := strings.ReplaceAll(stderrOf(err), user, s.Home+"/.ssh/config")
		switch {
		case strings.Contains(e, s.Home+"/.ssh/config.d/ws"):
			return fmt.Errorf("ssh refuses the old config.d/ws, which ~/.ssh/config also reads, as %s: "+
				"move it away, and run ws client macos apply again", e)
		case strings.Contains(e, "Bad configuration option"):
			return fmt.Errorf("ssh refuses ~/.ssh/config read after the new config.d/ws, as %s; "+
				"config.d/ws's IgnoreUnknown, read first, carries only ~/.ssh/config's own "+
				"patterns, so one for a file it includes must be there too", e)
		}
		return fmt.Errorf("ssh refuses ~/.ssh/config read after the new config.d/ws, as %s", e)
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
			return fmt.Errorf("ssh reads %s as %s: config.d/ws sets %s at %s, which another file "+
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
