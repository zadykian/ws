package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
	"github.com/zadykian/ws/roles/client/macos/internal/userfile"
)

// publicKey asks root's copy of ws for the Mac's public key. It returns "" where there is no copy
// yet, as a check before the first run, or where the key is to be made. It returns "" too where the
// copy, or a directory above it, is to change, as the sh fixed or replaced them before it ran the
// copy.
func (t *Tunnel) publicKey(ctx context.Context) (string, error) {
	if t.keyMade || t.untrusted {
		return "", nil
	}
	if _, err := t.M.RootAttrs(tunnelLibexec + "/ws"); errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	out, err := t.M.RunRoot(ctx, tunnelLibexec+"/ws", "client", "macos", "tunnel", "key")
	if err != nil {
		return "", fmt.Errorf("cannot read the Mac's public key: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// pubFile is the Mac's public key where the wizard's Tunnel stage reads it without sudo.
func (t *Tunnel) pubFile() string { return t.Home + "/.config/ws/tunnel.pub" }

// tell plans ~/.config/ws/tunnel.pub, once the key is made, and keeps the key for Finish.
func (t *Tunnel) tell(ctx context.Context) ([]step.Change, error) {
	public, err := t.publicKey(ctx)
	if err != nil || public == "" {
		return nil, err
	}
	t.public = public
	cs, link, err := userfile.WsDirs(t.M, t.Out, t.Home).Plan()
	if err != nil {
		return nil, err
	}
	pub, err := t.pub().Plan()
	if err != nil {
		return nil, err
	}
	cs = append(cs, pub...)
	link.Warn(t.Out, len(cs), "ws's files go there")
	return cs, nil
}

// pub is tunnel.pub, which holds the key that tell read. Its mode is the user's.
func (t *Tunnel) pub() userfile.File {
	return userfile.File{M: t.M, Out: t.Out, Path: t.pubFile(), Text: []byte(t.public + "\n")}
}

// publish writes tunnel.pub after the changes, where the key or root's copy was missing when the
// step planned, so that the next plan is empty.
func (t *Tunnel) publish(ctx context.Context) error {
	t.keyMade = false
	t.untrusted = false
	cs, err := t.tell(ctx)
	if err != nil {
		return err
	}
	for _, c := range cs {
		if err := change.Apply(ctx, t.M, t.Out, c); err != nil {
			return err
		}
	}
	return nil
}

// Finish prints the Mac's public key, with the commands that make it ws's peer, where the key is new
// or mac.conf has no WS_PUBLIC_KEY yet. A check prints nothing of it.
func (t *Tunnel) Finish(ctx context.Context) error {
	if t.Check || (!t.madeKey && t.Conf.PublicKey != "") {
		return nil
	}
	t.keyMade = false
	public, err := t.publicKey(ctx)
	if err != nil {
		return err
	}
	// /etc/wireguard and ws server ubuntu apply are root's, so a WS_USER other than root goes through
	// sudo, with a terminal for its password. The key, which is public, goes in the remote command, so
	// that ssh's stdin stays the terminal.
	sudo, tty := "", ""
	if t.Conf.User != "root" {
		sudo, tty = "sudo ", "-t "
	}
	ssh := "ssh " + tty + "-o HostName=ADDRESS -o Port=PORT ws"
	_, _ = fmt.Fprintf(t.Text, "The Mac's WireGuard public key is %s\nMake it ws's peer, mac, with a direct way to ws, "+
		"as ssh ws needs this done first. ADDRESS and PORT are that way:\n\n"+
		"    %s \"printf '%%s\\n' '%s' | %stee /etc/wireguard/mac.pub >/dev/null\"\n"+
		"    %s %sws server ubuntu apply --only wireguard\n", public, ssh, public, sudo, ssh, sudo)
	if t.Conf.PublicKey == "" {
		_, _ = io.WriteString(t.Text, "\nThen set WS_PUBLIC_KEY in ~/.config/ws/mac.conf to ws's key, which that run prints, and run\n"+
			"ws client macos apply again, which installs the tunnel's daemon.\n")
	}
	return nil
}
