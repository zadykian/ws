package tunnel

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"strconv"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/ufw"
	"github.com/zadykian/ws/ubuntu/values"
)

// Firewall holds the host's ufw rules of each guest's tunnel. They are its UDP port in, a route from
// its interface to Bridge, and no input on the interface, SSH included. A peer could reach the
// host's own sshd and ports through the interface, as ufw's SSH rule names none. So the deny
// sits first. A tunnel's rules carry comments of its
// own, as ufw's owner deletes any rule of its comment that it does not list. The port and the route
// of a tunnel that no guest holds any more are deleted, as are those of no other comment. Its deny
// stays: the interface and its peers stay, so the host stays shut to them.
type Firewall struct {
	M       effect.Machine
	Tunnels []values.GuestTunnel
	Bridge  string
}

// commentPrefix starts the comment of each guest tunnel's rules, which a slot's interface names.
// wg0, the host's own tunnel, is the role's spec's.
const commentPrefix = "tunnel wg"

func openComment(t values.GuestTunnel) string {
	return commentPrefix + strconv.Itoa(int(t.Slot)) + ": its port and its route"
}
func denyComment(t values.GuestTunnel) string {
	return commentPrefix + strconv.Itoa(int(t.Slot)) + denyEnding
}

// denyEnding ends the comment of each tunnel's deny of input.
const denyEnding = ": no input"

func (f Firewall) rules(t values.GuestTunnel) []ufw.Rules {
	port := ufw.Rule{Action: "allow", Proto: "udp", To: "any", ToPort: strconv.Itoa(t.Port), From: "any", FromPort: "any", Dir: "in"}
	deny := ufw.Rule{Action: "deny", Proto: "any", To: "any", ToPort: "any", From: "any", FromPort: "any", Dir: "in", Interface: t.Link()}
	return []ufw.Rules{
		{M: f.M, Comment: openComment(t), Want: []ufw.Rule{port, ufw.Route("allow", t.Link(), f.Bridge, "any", "")}},
		{M: f.M, Comment: denyComment(t), Want: []ufw.Rule{deny}, Front: true},
	}
}

// Plan plans each tunnel's rules, then the deletion of the port and route rules of a comment that no
// tunnel holds.
func (f Firewall) Plan(ctx context.Context) ([]resource.Change, error) {
	var wanted []string
	var changes []resource.Change
	for _, t := range f.Tunnels {
		for _, r := range f.rules(t) {
			wanted = append(wanted, r.Comment)
			cs, err := r.Plan(ctx)
			if err != nil {
				return nil, err
			}
			changes = append(changes, cs...)
		}
	}
	text, err := f.M.ReadRootFile(ufw.UserRules)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	var stale []string
	for _, r := range ufw.Parse(string(text)) {
		if strings.HasPrefix(r.Comment, commentPrefix) && !strings.HasPrefix(r.Comment, commentPrefix+"0:") &&
			!strings.HasSuffix(r.Comment, denyEnding) && !slices.Contains(wanted, r.Comment) && !slices.Contains(stale, r.Comment) {
			stale = append(stale, r.Comment)
		}
	}
	for _, c := range stale {
		cs, err := ufw.Rules{M: f.M, Comment: c}.Plan(ctx)
		if err != nil {
			return nil, err
		}
		changes = append(changes, cs...)
	}
	return changes, nil
}
