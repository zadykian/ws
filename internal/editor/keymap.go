// Package editor holds what the editors share, in no editor's terms: F12's keys, the settings, the
// file types and the language servers. An adapter of each editor renders them.
package editor

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"

	"go.yaml.in/yaml/v3"
)

//go:embed keymap/*.yaml
var keymapFiles embed.FS

// topics lists keymap/'s files in the order a load reads them. Scopes span files, and a scope's
// bindings keep this order, which is the order of Zed's file.
var topics = []string{
	"editing", "selection", "completion", "navigation", "search", "code", "folding", "problems",
	"files", "other",
}

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

var actionName = regexp.MustCompile(`^[a-z]+\.[a-z0-9-]+$`)

// LoadKeymap reads the embedded keymap. It fails on an unknown scope or field, on a misnamed
// action, and on a key that a scope binds twice.
func LoadKeymap() (Keymap, error) {
	keymap := Keymap{}
	for _, topic := range topics {
		name := "keymap/" + topic + ".yaml"
		data, err := keymapFiles.ReadFile(name)
		if err != nil {
			return nil, err
		}
		var file Keymap
		if err := decodeStrict(data, &file); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		for scope, bindings := range file {
			if !slices.Contains(Scopes, scope) {
				return nil, fmt.Errorf("%s: no scope %q", name, scope)
			}
			keymap[scope] = append(keymap[scope], bindings...)
		}
	}
	return keymap, keymap.validate()
}

func (k Keymap) validate() error {
	for scope, bindings := range k {
		seen := map[string]string{}
		for _, b := range bindings {
			if !actionName.MatchString(b.Action) {
				return fmt.Errorf("%s: misnamed action %q", scope, b.Action)
			}
			for _, chord := range b.Keys {
				key := chord.String()
				if other, taken := seen[key]; taken {
					return fmt.Errorf("%s: %s binds both %s and %s", scope, key, other, b.Action)
				}
				seen[key] = b.Action
			}
		}
	}
	return nil
}

// decodeStrict fails on a field the type lacks, so that a misspelt one is no silent loss.
func decodeStrict(data []byte, out any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(out); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
