// Package stage holds the names of the Mac's wizard stages, the Mac as their side, and the change
// that a stage reports where it waits. The stages are in the packages beside it; they share these
// names so that none imports another for one.
package stage

import guided "github.com/zadykian/ws/engine/core/wizard"

// The stages' names, which their messages and the wizard's screens show.
const (
	// Update is the stage of ws's binary on the Mac.
	Update = "ws on the Mac"
	// Tools is the stage of the Command Line Tools.
	Tools = "Tools"
	// Homebrew is the stage of Homebrew.
	Homebrew = "Homebrew"
	// Answers is the stage that gives mac.conf the values it lacks.
	Answers = "Answers"
	// Key is the stage of the Mac's key.
	Key = "Mac key"
	// GitHub is the stage of gh on the Mac.
	GitHub = "gh on the Mac"
	// Setup is the stage that applies the Mac's steps.
	Setup = "ws client macos apply"
	// Host is the stage of ws's host key.
	Host = "Host key"
	// Login is the stage that checks the login on ws.
	Login = "Login"
	// Sudo is the stage that checks sudo on ws.
	Sudo = "Sudo"
	// Install is the stage of ws's binary on ws.
	Install = "ws on ws"
	// Cable is the stage of the direct link.
	Cable = "Cable"
	// Tunnel is the stage of the tunnel.
	Tunnel = "Tunnel"
)

// Mac is the Mac, one of the wizard's two machines, and the title of its list. The other is
// session.Ws.
// A stage that changes both is the Mac's where its last change lands there.
// The Tunnel stage is one: its last write is WS_PUBLIC_KEY in the server's file.
const Mac guided.Side = "Client (macOS)"

// Todo is the change that a stage reports where it waits on another.
type Todo string

// String is the change's text.
func (c Todo) String() string { return string(c) }
