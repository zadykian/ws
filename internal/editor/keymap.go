// Package editor holds what the editors share, in no editor's terms: F12's keys, the settings, the
// file types and the language servers. An adapter of each editor renders them.
package editor

// Scope says where a binding holds.
type Scope string

// The scopes, from the widest.
const (
	Global    Scope = "global"    // wherever focus is
	Text      Scope = "text"      // every text input, the one-line ones too
	Code      Scope = "code"      // the full editor alone
	Signature Scope = "signature" // the signature popup
	Tree      Scope = "tree"      // the file tree
	Git       Scope = "git"       // the git panel
	Commit    Scope = "commit"    // the commit message
	Debug     Scope = "debug"     // a stopped debug session
	Panel     Scope = "panel"     // any panel but the terminal
	Terminal  Scope = "terminal"  // the terminal
)

// Scopes lists the scopes.
var Scopes = []Scope{Global, Text, Code, Signature, Tree, Git, Commit, Debug, Panel, Terminal}

// Binding gives an action its keys. F12 names the IDEs' actions that it comes from.
type Binding struct {
	Action string   `yaml:"action"`
	Keys   []Chord  `yaml:"keys"`
	F12    []string `yaml:"f12"`
}

// Keymap holds each scope's bindings, in the order of topics.
type Keymap map[Scope][]Binding
