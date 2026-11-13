package widget

import "charm.land/bubbles/v2/key"

// KeyMap is the keys that hold the same meaning on every screen.
type KeyMap struct {
	// Next and Prev move the focus.
	Next key.Binding
	Prev key.Binding
	// Press presses the button in focus.
	Press key.Binding
	// Up, Down, PageUp, PageDown, Top and Bottom move through a list or a window of output.
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Top      key.Binding
	Bottom   key.Binding
	// Quit leaves a screen. Interrupt is Ctrl-C, which raw mode turns into a key.
	Quit      key.Binding
	Interrupt key.Binding
}

// Keys is the map that the screens read. A screen that disables a button does so on its own copy.
// PageUp and PageDown take a control key too, as Terminal.app keeps those two keys for its own
// scrollback.
var Keys = KeyMap{
	Next:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next")),
	Prev:      key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
	Press:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "press")),
	Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑", "up")),
	Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓", "down")),
	PageUp:    key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pgup", "page up")),
	PageDown:  key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pgdn", "page down")),
	Top:       key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("home", "top")),
	Bottom:    key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("end", "bottom")),
	Quit:      key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Interrupt: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "stop")),
}
