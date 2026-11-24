package tunnel

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/ubuntu/values"
)

//go:embed 60-guest.netdev.tmpl 60-guest.network.tmpl
var linkTemplates embed.FS

// networkdDir holds the files that networkd reads.
const networkdDir = "/etc/systemd/network"

// peerData is a peer of a guest's tunnel, with the key of its file.
type peerData struct{ Name, Key, Address string }

// linkData is what the templates of a guest's tunnel read.
type linkData struct {
	Link, Guest, KeyFile, PeersDir, SlotAddress, Subnet string
	Port                                                int
	Peers                                               []peerData
}

// Links settles a guest's WireGuard interface on the host for each tunnel. It writes the .netdev
// and the .network, then reloads networkd where a file changed or an interface lacks the file's
// key. An interface of another key is deleted first, as networkd reads the key as it creates it. A
// peer without its key file is left out. The keys are made before, by Key. Tries and Pause bound the
// wait for each new interface, and Sleep stands for time.Sleep where nil.
type Links struct {
	M       effect.Machine
	Out     effect.Out
	Tunnels []values.GuestTunnel
	Tries   int
	Pause   time.Duration
	Sleep   func(time.Duration)
}

// data reads each peer's key and gives the templates' data.
func (l Links) data(t values.GuestTunnel) (linkData, error) {
	d := linkData{Link: t.Link(), Guest: t.Guest.Name, KeyFile: t.KeyFile(), PeersDir: t.PeersDir(), Port: t.Port,
		SlotAddress: t.SlotAddress().String(), Subnet: t.Slot.Net().String()}
	for _, p := range t.Peers {
		text, err := l.M.ReadRootFile(t.PeersDir() + "/" + p.Name + ".pub")
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return linkData{}, err
		}
		d.Peers = append(d.Peers, peerData{p.Name, strings.TrimSpace(string(text)), p.Address})
	}
	return d, nil
}

// files returns a tunnel's two files: the .netdev that networkd reads as the group systemd-network,
// and the .network.
func (l Links) files(t values.GuestTunnel, gid int) ([]file.File, error) {
	d, err := l.data(t)
	if err != nil {
		return nil, err
	}
	netdev, err := resource.Render(linkTemplates, "60-guest.netdev.tmpl", d)
	if err != nil {
		return nil, err
	}
	network, err := resource.Render(linkTemplates, "60-guest.network.tmpl", d)
	if err != nil {
		return nil, err
	}
	return []file.File{
		{M: l.M, Path: networkdDir + "/60-" + t.Link() + ".netdev", Content: netdev, Mode: 0o640, GID: gid},
		{M: l.M, Path: networkdDir + "/60-" + t.Link() + ".network", Content: network, Mode: 0o644},
	}, nil
}

// Plan plans the files, then the deletion of each interface that holds another key. One reload of
// networkd follows where any file or interface is due, then the wait for each interface made.
func (l Links) Plan(ctx context.Context) ([]resource.Change, error) {
	gid, err := groupID(l.M)
	if err != nil {
		return nil, err
	}
	var changes, deletes []resource.Change
	var waits []waitForKey
	var why []string
	for _, t := range l.Tunnels {
		files, err := l.files(t, gid)
		if err != nil {
			return nil, err
		}
		changed := false
		for _, f := range files {
			cs, err := f.Plan(ctx)
			if err != nil {
				return nil, err
			}
			if len(cs) > 0 && !changed {
				changed = true
				why = append(why, t.Link()+"'s files change")
			}
			changes = append(changes, cs...)
		}
		w := WG0{M: l.M, Out: l.Out, Tries: l.Tries, Pause: l.Pause, Sleep: l.Sleep, Name: t.Link(), File: t.KeyFile()}
		cs, err := w.Plan(ctx)
		if err != nil {
			return nil, err
		}
		for _, c := range cs {
			switch c := c.(type) {
			case deleteLink:
				deletes = append(deletes, c)
			case waitForKey:
				waits = append(waits, c)
			case reloadNetworkd:
				why = append(why, c.why)
			}
		}
	}
	if len(why) == 0 {
		return changes, nil
	}
	changes = append(changes, deletes...)
	changes = append(changes, reloadNetworkd{l.M, "the guests' tunnels", strings.Join(why, ", ")})
	for _, w := range waits {
		changes = append(changes, w)
	}
	return changes, nil
}
