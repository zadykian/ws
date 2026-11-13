package stages

import (
	"strings"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// extra is what mac.conf needs besides its target.
var extra = map[string]string{"cloud": "WS_REMOTE=ws.example.org\nWS_REMOTE_PORT=22022\n"}

// idleRun is ws's status after a full run of this ws, v1.2.3, that exited 0.
var idleRun = session.Status{Run: session.Stamp{Version: "v1.2.3", Exit: "0"}}

// fileReach is a session.Reach over Ws on M, whose wants are the keys that M's /h/mac.conf names.
type fileReach struct {
	sessiontest.Reach
}

func (fileReach) Rerun() string { return "run the wizard with --update" }

func (r fileReach) Want() (session.Want, error) {
	var w session.Want
	for _, line := range strings.Split(string(r.M.Files["/h/mac.conf"]), "\n") {
		if v, ok := strings.CutPrefix(line, "WS_TARGET="); ok {
			w.Target = v
		}
		if v, ok := strings.CutPrefix(line, "WS_USER="); ok {
			w.User = v
		}
	}
	return w, nil
}

// newRun returns the stage over a Machine whose mac.conf holds the target, and whose remote script
// answers status with the value's text.
func newRun(target string, status session.Status, update, check bool) (*WsRun, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=" + target + "\nWS_USER=" + values.Default().User + "\n" + extra[target])
	out := &effecttest.Out{}
	ws := session.New(fileReach{sessiontest.Reach{M: m}}, session.Scripts{Remote: []byte("x"), Files: []byte("y")})
	m.Results[sessiontest.Op("status")] = effecttest.Result{Out: sessiontest.Text(status)}
	return &WsRun{Ws: ws, Check: check, Out: out, Version: "v1.2.3", Update: update, Await: "ws on ws"}, m
}
