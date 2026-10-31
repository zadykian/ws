package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

// validate stops where ssh refuses the new config.d/ws, alone or as ssh will read it after the
// run: first the new file, then ~/.ssh/config, then the system's file. ~/.ssh/config's Include of
// config.d/ws is left out, as it would read the file that the run replaces. It writes its files in
// a directory of its own, and removes it.
func (s SSH) validate(ctx context.Context, f sshFacts, text []byte) error {
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
	for _, host := range []string{"ws-ssh", "ws"} {
		if _, err := s.M.Run(ctx, "ssh", "-G", "-F", next, host); err != nil {
			return fmt.Errorf("ssh refuses the config it would write, as %s; Match localnetwork needs OpenSSH 9.4",
				stderrOf(err))
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
	for _, host := range []string{"ws", "ws-ssh"} {
		if _, err := s.M.Run(ctx, "ssh", "-G", "-F", all, host); err != nil {
			e := strings.ReplaceAll(stderrOf(err), user, s.Home+"/.ssh/config")
			switch {
			case strings.Contains(e, s.Home+"/.ssh/config.d/ws"):
				return fmt.Errorf("ssh refuses the old config.d/ws, which ~/.ssh/config also reads, as %s: "+
					"move it away, and run setup.sh again", e)
			case strings.Contains(e, "Bad configuration option"):
				return fmt.Errorf("ssh refuses ~/.ssh/config read after the new config.d/ws, as %s; "+
					"config.d/ws's IgnoreUnknown, read first, carries only ~/.ssh/config's own "+
					"patterns, so one for a file it includes must be there too", e)
			}
			return fmt.Errorf("ssh refuses ~/.ssh/config read after the new config.d/ws, as %s", e)
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
