package link

import (
	"context"
	"errors"
	"fmt"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"strings"
	"time"
)

// handshake waits up to 30 s for ws-tunnel's handshake, where the run reached ws at its ways. Then
// it logs in to ws through the tunnel once, by the key that ws takes, as fail2ban on ws bans
// the Mac's end after a few that fail. ws's new key reaches ws-tunnel by ws client macos apply's SIGHUP.
// The Mac's new peer reaches ws by a restart, so that it handshakes at once. Without a
// handshake, it adds the tunnel to the closing list, and the run goes on, as the later stages reach
// ws by a direct way, outside the tunnel.
func (t *Tunnel) handshake(ctx context.Context, c conf.Conf, kick bool) error {
	end, mac := c.TunnelSlot().Server(), c.TunnelSlot().Mac()
	if first := t.SSH.First(); first != "" {
		once := "server.conf's way leads to " + t.SSH.Host
		if c.Target == "lan" {
			once = t.SSH.Host + "'s lease is static"
		}
		t.Hand.Add("The tunnel's handshake: ws-tunnel goes to %s's ways, not %s, where this run reached "+
			"%s; once %s, run the wizard again, which checks it.", t.SSH.Host, first, t.SSH.Host, once)
		return nil
	}
	if kick {
		if err := t.M.Pass(ctx, true, "launchctl", "kickstart", "-k", "system/"+t.label()); err != nil {
			return fmt.Errorf("%s: launchctl kickstart -k system/%s failed", stage.Tunnel, t.label())
		}
		t.Out.Say("%s: ran launchctl kickstart -k system/%s, for wg0's new peer on ws", stage.Tunnel, t.label())
	}
	t.Sleep(2 * time.Second)
	if !t.shake(ctx, 30) {
		t.none = true
		t.Hand.Add("The tunnel's handshake: ws-tunnel had none with %s within 30 s, so ssh %s fails while "+
			"a direct way works. %s must take UDP %s, through the router's forward or a cloud's firewall "+
			"where WS_REMOTE leads through one, and ws-tunnel must stay allowed in the Background "+
			"(roles/client/macos/README.md, The Mac's tunnel). Then run the wizard again, which checks it.", t.SSH.Host, t.SSH.Host, t.SSH.Host, c.TunnelPort())
		return nil
	}
	if !t.via(ctx, end) {
		return errors.New(stage.Tunnel + ": ssh -G " + t.SSH.Host + " gives another host than " + end.String() + ", ws's end of the " +
			"tunnel, which ws client macos apply's config.d/ws names")
	}
	out, err := t.SSH.Tunnel(ctx, "-o", "BatchMode=yes", t.SSH.Host, "echo", "ok")
	if err != nil || strings.TrimSpace(string(out)) != "ok" {
		last := ""
		if err != nil {
			lines := strings.Split(strings.TrimSpace(remote.ExitStatus.ReplaceAllString(err.Error(), "")), "\n")
			last = ": " + strings.TrimSpace(lines[len(lines)-1])
		}
		return errors.New(stage.Tunnel + ": ssh " + t.SSH.Host + " failed through the tunnel, where ws's run lets SSH in on " +
			"wg0, and fail2ban bans " + mac.String() + " for an hour after 5 failures" + last)
	}
	return nil
}
