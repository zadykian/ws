package ssh

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/sshcfg"
	"github.com/zadykian/ws/macos/internal/keys"
	"github.com/zadykian/ws/macos/roles/cld"
)

// Text returns /etc/ws/ssh_config: a block for each of hosts, which offers key. It stops where a Host
// of ~/.ssh/config names a server. It stops where ssh refuses the text, alone or read whole with
// ~/.ssh/config and the system's file, so that a bootstrap stops before any change. It reads as the
// user, and writes in a scratch directory alone. With no host there is nothing for ssh to read.
func Text(ctx context.Context, m effect.Machine, home string, key keys.NamedKey, hosts []sshcfg.Host) (string, error) {
	u, err := readUserConfig(m, home)
	if err != nil {
		return "", err
	}
	if err := ownHost(u.Text, hosts); err != nil {
		return "", err
	}
	text := sshcfg.Config(sshcfg.Inputs{
		Hosts:       hosts,
		Patterns:    sshcfg.IgnoreUnknown(u.Text, "UseKeychain"),
		IdentityPub: key.Pub, IdentityAgent: key.Agent, CldTabs: tabsAhead(ctx, m, home),
	})
	if len(hosts) == 0 {
		return text, nil
	}
	if err := validate(ctx, m, home, u, hosts, []byte(text)); err != nil {
		return "", err
	}
	return text, nil
}

// readUserConfig reads ~/.ssh/config, through a link, and where its Include of /etc/ws/ssh_config
// is. A missing file is none.
func readUserConfig(m effect.Machine, home string) (userConfig, error) {
	path := home + "/.ssh/config"
	text, err := m.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return userConfig{}, nil
	}
	if err != nil {
		return userConfig{}, fmt.Errorf("cannot read %s: %w", path, err)
	}
	u := userConfig{Exists: true, Text: string(text)}
	_, u.IncludeLine = sshcfg.IncludeAt(u.Text, sshcfg.File)
	return u, nil
}

// tabsAhead is whether the cld that cld's unit leaves in ~/.local/bin will have restore tabs, where
// Ghostty's app is there. The unit installs a missing cld and updates a release to the latest, both
// of which have it. A cld built from a clone stays, so its own help says. So the text
// is the same before cld's unit runs and after, and a second bootstrap changes nothing.
func tabsAhead(ctx context.Context, m effect.Machine, home string) bool {
	there := false
	for _, app := range []string{"/Applications/Ghostty.app", home + "/Applications/Ghostty.app"} {
		if info, err := m.Stat(app); err == nil && info.Mode.IsDir() {
			there = true
		}
	}
	if !there {
		return false
	}
	bin := home + "/.local/bin/cld"
	if info, err := m.Stat(bin); err != nil || info.Mode&0o111 == 0 {
		return true
	}
	if out, err := m.Run(ctx, bin, "--version"); err == nil && cld.IsRelease(string(out)) {
		return true
	}
	_, err := m.Run(ctx, bin, "help", "restore", "tabs")
	return err == nil
}
