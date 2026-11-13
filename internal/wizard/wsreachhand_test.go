package wizard

import (
	"context"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"slices"
	"strings"
	"testing"
)

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
		far := ws.reach.(fakeReach)
		r := far.Reach.(WsReach)
		r.Read = func() (Want, error) { return Want{Target: tc.target}, nil }
		far.Reach = r
		ws.reach = far
		h := &guided.Hand{}
		ws.Hand(context.Background(), h)
		var got []string
		for _, l := range h.Lines() {
			head, _, _ := strings.Cut(l, ":")
			got = append(got, head)
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: Hand = %q; want %q", name, h.Lines(), tc.want)
		}
	}
}
