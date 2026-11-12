package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/sshcfg"
)

// daemonLabel is the LaunchDaemon of the tunnel.
const daemonLabel = "com.github.zadykian.ws-tunnel"

// PickKey takes the key that mac.conf names for ws's login, and checks its .pub. A key that
// Secretive's agent lists needs no file: it comes as ws's copy, with the copy's text, which a later
// step writes. Without a key named, it stops and names a line for each of the agent's keys.
func PickKey(ctx context.Context, m effect.Machine, c conf.Conf, home string) (sshcfg.Identity, error) {
	// An agent that does not answer leaves a named .pub to read, and a stop with no line to add.
	keys, _, _ := SecretiveKeys(ctx, m, home)
	if c.IdentityPubKey == "" {
		return sshcfg.Identity{}, unnamed("WS_IDENTITY_PUB_KEY", "ws's login key", keys)
	}
	id := sshcfg.NamedIdentity(c.IdentityPubKey, c.IdentityAgent, home)
	file := sshcfg.Expand(id.Pub, home)
	if k, ok := held(keys, file, home); ok {
		id.Pub, id.Text = k.Pub(), k.Line+"\n"
		_, err := sshcfg.ParsePub(k.Line)
		return id, refused(sshcfg.Expand(id.Pub, home), "WS_IDENTITY_PUB_KEY", err)
	}
	text, err := m.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return id, fmt.Errorf("%s, mac.conf's WS_IDENTITY_PUB_KEY, is missing: make the key first, "+
			"as docs/mac.md says", file)
	}
	if errors.Is(err, fs.ErrPermission) && sshcfg.SecretiveKey(file, home) {
		return id, fmt.Errorf("cannot read %s, and Secretive's agent lists no key of it: open Secretive, "+
			"or name another in mac.conf's WS_IDENTITY_PUB_KEY", file)
	}
	if err != nil {
		return id, fmt.Errorf("cannot read %s: make it readable, or name another in mac.conf's WS_IDENTITY_PUB_KEY", file)
	}
	if _, err := sshcfg.ParsePub(string(text)); err != nil {
		return id, refused(file, "WS_IDENTITY_PUB_KEY", err)
	}
	if _, err := m.Run(ctx, "ssh-keygen", "-l", "-f", file); err != nil {
		return id, fmt.Errorf("ssh-keygen cannot read %s: name the .pub of a P-256 or ed25519 key in "+
			"mac.conf's WS_IDENTITY_PUB_KEY", file)
	}
	id.Text = string(text)
	return id, nil
}

// refused is the stop where the .pub of file is no key that ws takes, err being why. The remedy
// names the variable of mac.conf that names the file. A nil err passes.
func refused(file, name string, err error) error {
	var typ sshcfg.TypeError
	switch {
	case err == nil:
		return nil
	case errors.Is(err, sshcfg.ErrCertificate):
		return fmt.Errorf("%s %w: name its key's own .pub in mac.conf's %s", file, err, name)
	case errors.As(err, &typ):
		return fmt.Errorf("%s %w: name the .pub of a P-256 or ed25519 key in mac.conf's %s", file, err, name)
	}
	return fmt.Errorf("%s %w", file, err)
}

// cldTabs is whether Ghostty's app is there and the cld in ~/.local/bin has restore tabs. With it,
// ssh never runs a cld that would refuse restore tabs --watch at each connection.
func (s SSH) cldTabs(ctx context.Context) bool {
	there := false
	for _, app := range []string{"/Applications/Ghostty.app", s.Home + "/Applications/Ghostty.app"} {
		if info, err := s.M.Stat(app); err == nil && info.Mode.IsDir() {
			there = true
		}
	}
	if !there {
		return false
	}
	cld := s.Home + "/.local/bin/cld"
	info, err := s.M.Stat(cld)
	if err != nil || info.Mode&0o111 == 0 {
		return false
	}
	_, err = s.M.Run(ctx, cld, "help", "restore", "tabs")
	return err == nil
}
