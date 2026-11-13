// Package wizard holds the stages of ws client macos wizard and ws's, each a step.Step. The guided
// run, its lock and its list of what stays by hand are in engine/core/wizard.
package wizard

import guided "github.com/zadykian/ws/engine/core/wizard"

// OnMac is the Mac, one of the wizard's two machines. The other is session.Ws.
// A stage that changes both is the Mac's where its last change lands there.
// The Tunnel stage is one: its last write is WS_PUBLIC_KEY in mac.conf.
const OnMac guided.Side = "Mac"
