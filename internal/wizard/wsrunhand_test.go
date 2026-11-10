package wizard

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/conf"
)

func TestMacHand(t *testing.T) {
	for name, tc := range map[string]struct {
		target string
		status wsStatus
		want   string
	}{
		"lan":   {"lan", wsStatus{FansRefused: true}, "ws's fans' driver"},
		"local": {"lan", wsStatus{}, "macOS's local network"},
		"cloud": {"cloud", wsStatus{OutboundOff: true}, "The provider's firewall: UDP 51820"},
		"off":   {"lan", wsStatus{OutboundOff: true}, "ws's outbound table"},
	} {
		r, _ := newRun(tc.target, tc.status, false, false)
		h := &Hand{}
		r.Ws.Hand(context.Background(), h)
		if got := strings.Join(h.lines, "\n"); !strings.Contains(got, tc.want) {
			t.Errorf("%s: Hand = %q; want %q", name, got, tc.want)
		}
	}
}

// ws's own list holds ws's lines alone, with nothing of the Mac's, and each device's key.
func TestWsHand(t *testing.T) {
	device := "Each device's ssh key on ws"
	for name, tc := range map[string]struct {
		target string
		status wsStatus
		want   []string
	}{
		"fans":  {"cloud", wsStatus{FansRefused: true}, []string{"ws's fans' driver", device}},
		"lan":   {"lan", wsStatus{}, []string{"ws's lease, on a machine on the LAN", device}},
		"cloud": {"cloud", wsStatus{}, []string{device}},
	} {
		ws, _ := wsSession(tc.status)
		r := ws.reach.(WsReach)
		r.Want = func() (conf.Conf, error) { return conf.Conf{Target: tc.target}, nil }
		ws.reach = r
		h := &Hand{}
		ws.Hand(context.Background(), h)
		var got []string
		for _, l := range h.lines {
			head, _, _ := strings.Cut(l, ":")
			got = append(got, head)
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: Hand = %q; want %q", name, h.lines, tc.want)
		}
	}
}
