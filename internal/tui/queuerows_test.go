package tui

import "github.com/zadykian/ws/step"

// followed is l after e, the cursor aside.
func followed(l pickList, e step.Event) pickList {
	l, _ = l.follow(e)
	return l
}
