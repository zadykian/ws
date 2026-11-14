package link

import (
	"context"
	"testing"

	"github.com/zadykian/ws/roles/server/ubuntu/session"
)

func TestCablePlanAsksRunDoneForTheTunnelPair(t *testing.T) {
	c, m, _ := newCable(t, session.Status{}, cableLink, nil)
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\nWS_SLOT=2\nWS_PORT=51999\n" + wayConf)
	c.Check = true
	var got session.Want
	c.RunDone = func(_ context.Context, w session.Want) bool { got = w; return false }
	if _, err := c.Plan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !got.Address.IsValid() || got.Port != "51999" {
		t.Errorf("RunDone got %+v; want the tunnel's address and port 51999", got)
	}
}
