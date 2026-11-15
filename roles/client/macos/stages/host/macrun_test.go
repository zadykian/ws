package host

import (
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	macvalues "github.com/zadykian/ws/roles/client/macos/values"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// extra is what values.yml needs besides its target.
var extra = map[string]string{"cloud": macvalues.Text("WS_REMOTE", "ws.example.org", "WS_REMOTE_PORT", "22022")}

// newRun returns ws's run over a Machine whose values.yml holds the target, and whose remote script
// answers status with the value's text.
func newRun(target string, status session.Status, update, check bool) (*stages.WsRun, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Files["/h/values.yml"] = []byte(macvalues.Text("WS_TARGET", target, "WS_USER", values.Default().User) + extra[target])
	out := &effecttest.Out{}
	ws := macSession(m, remote.SSH{Host: "ws", M: m, Lock: "/l", Ways: MacWays{M: m, Conf: "/h/values.yml"}}, out, status)
	return &stages.WsRun{Ws: ws, Check: check, Out: out, Version: "v1.2.3", Update: update, Await: stage.Install}, m
}
