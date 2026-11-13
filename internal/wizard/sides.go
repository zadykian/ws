// Package wizard holds the stages of ws client macos wizard and ws's, each a step.Step. The guided
// run, its lock and its list of what stays by hand are in engine/core/wizard.
package wizard

import guided "github.com/zadykian/ws/engine/core/wizard"

// The two machines of the wizard. A stage that changes both is the Mac's where its last change
// lands on the Mac, as in the Tunnel stage, whose last write is WS_PUBLIC_KEY in mac.conf.
const (
	OnMac guided.Side = "Mac"
	OnWs  guided.Side = "ws"
)
