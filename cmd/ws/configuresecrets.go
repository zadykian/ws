package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"

	"github.com/zadykian/ws/internal/keychain"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
)

// secretsLabels are the entries that the form always offers, besides those the file holds.
var secretsLabels = []string{
	secrets.KindToken + " github", secrets.KindToken + " azure_devops", secrets.KindPassword + " vpn_p12",
}

// secretsSheet is the secrets file on the Configure screen. held lists its entries as "KIND NAME",
// never a value: a field says whether its entry is set, and starts empty. A file entry holds bytes,
// and an env or git entry beyond these needs a name, so ws mac secrets sets those.
func secretsSheet(side secretsSide, held []string) tui.Sheet {
	return tui.Sheet{
		Name: "secrets",
		Fields: func(map[string]string) []tui.FieldSpec {
			labels := slices.Clone(secretsLabels)
			for _, k := range held {
				if kind, _, _ := strings.Cut(k, " "); kind == secrets.KindEnv || kind == secrets.KindGit {
					labels = append(labels, k)
				}
			}
			specs := make([]tui.FieldSpec, 0, len(labels))
			for _, l := range labels {
				help := "not set: type a value to set it"
				if slices.Contains(held, l) {
					help = "set: type a value to replace it"
				}
				specs = append(specs, tui.FieldSpec{Kind: tui.PasswordField, Label: l, Help: help})
			}
			return specs
		},
		// Save sets each entry in turn. A failed one leaves those before it set, and the sheet reads
		// what the file then holds.
		Save: func(ctx context.Context, _ *tui.Bridge, edits map[string]string) (tui.Sheet, error) {
			var err error
			for _, label := range slices.Sorted(maps.Keys(edits)) {
				if err = side.set(ctx, label, edits[label]); err != nil {
					break
				}
			}
			keys, kerr := side.keys(ctx)
			if kerr != nil {
				keys = held
			}
			return secretsSheet(side, keys), errors.Join(err, kerr)
		},
	}
}

// secretsSide is where the secrets sheet reads and sets the file's entries, by label: "KIND NAME".
type secretsSide interface {
	keys(ctx context.Context) ([]string, error)
	set(ctx context.Context, label, value string) error
}

// storeSide is the Mac's secrets file.
type storeSide struct{ store secretsStore }

func (s storeSide) set(ctx context.Context, label, value string) error {
	return s.store.edit(ctx, func(cur *secrets.Secrets) error { return setEntry(cur, label, value) })
}

func (s storeSide) keys(ctx context.Context) ([]string, error) { return heldSecrets(ctx, s.store) }

// setEntry sets the entry that a label names. A git entry keeps its user name.
func setEntry(s *secrets.Secrets, label, value string) error {
	kind, name, _ := strings.Cut(label, " ")
	if kind == secrets.KindGit {
		g := s.Git[name]
		g.Token = value
		s.SetGit(name, g)
		return nil
	}
	return s.Set(kind, name, value)
}

// heldSecrets lists the entries of the secrets file as "KIND NAME". A Keychain without the item, or a
// missing file, holds none: a save then tells what to run.
func heldSecrets(ctx context.Context, store secretsStore) ([]string, error) {
	id, err := store.k.Identity(ctx)
	if errors.Is(err, keychain.ErrNoItem) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the secrets: %w", err)
	}
	cur, err := secrets.LoadRaw(store.m, store.path, id)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return cur.Keys(), nil
}

// wsSide is ws's secrets file, which ws server secrets edits on ws over the wizard's ssh, a value on
// ssh's stdin. set git, with no flag, keeps the entry's user name.
type wsSide struct{ ws *wizard.Remote }

func (s wsSide) set(ctx context.Context, label, value string) error {
	kind, name, _ := strings.Cut(label, " ")
	_, err := s.ws.Secrets(ctx, []byte(value+"\n"), "set", kind, "--", name)
	return err
}

func (s wsSide) keys(ctx context.Context) ([]string, error) {
	out, err := s.ws.Secrets(ctx, nil, "keys")
	if err != nil {
		return nil, fmt.Errorf("the secrets on ws: %w", err)
	}
	return strings.FieldsFunc(string(out), func(r rune) bool { return r == '\n' }), nil
}
