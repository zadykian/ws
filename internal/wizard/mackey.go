package wizard

import (
	"time"

	"github.com/zadykian/ws/engine/core/effect"
)

// macKey names the stage in its messages.
const macKey = "Mac key"

// MacKey is the stage that names the key that ssh offers ws in mac.conf's WS_IDENTITY_PUB_KEY.
// Where mac.conf names none, it offers each P-256 key of Secretive's agent, and a new one, which
// the user makes in Secretive's app. It never makes a key in a file, and never takes a key unasked:
// a key made for another use would go to GitHub.
type MacKey struct {
	// Conf is mac.conf's path, Home the user's, and Lock the run's lock, whose directory holds the
	// link that fingerprints a private key.
	Conf, Home, Lock string
	// Tty tells whether a terminal is there to ask.
	Tty   bool
	M     effect.Machine
	P     effect.Prompt
	Out   effect.Out
	Sleep func(time.Duration)
}

// Name is the stage's name.
func (*MacKey) Name() string { return macKey }

// Side is the Mac: its ssh key.
func (*MacKey) Side() Side { return OnMac }

// WaitsOn is Answers, which gives mac.conf the lines that this stage reads.
func (*MacKey) WaitsOn() string { return answers }
