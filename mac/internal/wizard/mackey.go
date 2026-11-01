package wizard

import (
	"time"

	"github.com/zadykian/ws/mac/internal/effect"
)

// macKey names the stage in its messages.
const macKey = "Mac key"

// MacKey is the stage that makes the key that ssh offers ws, as setup.PickKey finds it from
// mac.conf. Without WS_IDENTITY, the key is Secretive's, in the Mac's Secure Enclave, unless the
// answer at the terminal is no: then ssh-keygen makes ~/.ssh/id_ed25519. It writes the key into
// mac.conf as WS_IDENTITY, so that a Secretive key made later does not take the file key's place.
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

// WaitsOn is Answers, which gives mac.conf the lines that this stage reads.
func (*MacKey) WaitsOn() string { return answers }
