package session_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
)

// ws's own list holds ws's lines alone, with nothing of the Mac's, and each device's key.
func TestWsHand(t *testing.T) {
	device := "Each device's ssh key on ws"
	for name, tc := range map[string]struct {
		target string
		status session.Status
		want   []string
	}{
		"fans":  {"cloud", session.Status{FansRefused: true}, []string{"ws's fans' driver", device}},
		"lan":   {"lan", session.Status{}, []string{"ws's lease, on a machine on the LAN", device}},
		"cloud": {"cloud", session.Status{}, []string{device}},
	} {
		m := effecttest.NewMachine()
		read := func() (session.Want, error) { return session.Want{Target: tc.target}, nil }
		reach := session.WsReach{M: m, Read: read, Here: m, Out: &effecttest.Out{}}
		ws := session.New(sessiontest.Reach{Reach: reach, M: m}, session.Scripts{Remote: []byte("x")})
		m.Results[sessiontest.Op("status")] = effecttest.Result{Out: sessiontest.Text(tc.status)}
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
