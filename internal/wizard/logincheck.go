package wizard

import (
	"github.com/zadykian/ws/engine/core/effect"
)

// loginStage names the stage in its messages.
const loginStage = "Login"

// Login is the stage that checks that WS_USER on ws takes the Mac's key, so that ssh logs in to ws.
// The wizard adds no key to ws. ws's keys come from the GitHub account at its install, and the gh
// on the Mac stage has put the Mac's there. So the stage stops where WS_USER refuses the key, and
// says what to do.
type Login struct {
	// SSH is the Mac's ssh to ws. Ws is the session to ws, whose status the stage reads.
	SSH SSH
	Ws  *Session
	// Gh names the Mac's key in the stop.
	Gh GhCLI
	// Tty tells whether a terminal is there to ask ws's address, where ws doesn't answer at its way.
	// Check is set under --check, which asks nothing.
	Tty, Check bool
	// Env looks up a variable of the environment.
	Env func(string) (string, bool)
	P   effect.Prompt
	Out effect.Out
}

// Name is the stage's name.
func (*Login) Name() string { return loginStage }

// Side is the Mac: the stage checks the Mac's login and changes nothing on ws.
func (*Login) Side() Side { return OnMac }

// WaitsOn is the Host key stage, without which ssh logs in nowhere.
func (*Login) WaitsOn() string { return hostStage }
