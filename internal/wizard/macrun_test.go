package wizard

import (
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// extra is what mac.conf needs besides its target.
var extra = map[string]string{"cloud": "WS_REMOTE=ws.example.org\nWS_REMOTE_PORT=22022\n"}

// newRun returns ws's run over a Machine whose mac.conf holds the target, and whose remote script
// answers status with the value's text.
func newRun(target string, status session.Status, update, check bool) (*stages.WsRun, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=" + target + "\nWS_USER=" + values.Default().User + "\n" + extra[target])
	out := &effecttest.Out{}
	ws := macSession(m, remote.SSH{M: m, Lock: "/l", Ways: host.MacWays{M: m, Conf: "/h/mac.conf"}}, out, status)
	return &stages.WsRun{Ws: ws, Check: check, Out: out, Version: "v1.2.3", Update: update, Await: stage.Install}, m
}
