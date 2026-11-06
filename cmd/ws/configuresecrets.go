package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/zadykian/ws/internal/keychain"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/tui"
)

// secretsLabels are the entries that the form always offers, besides those the file holds.
var secretsLabels = []string{
	secrets.KindToken + " github", secrets.KindToken + " azure_devops", secrets.KindPassword + " vpn_p12",
}

// secretsSheet is the secrets file on the Configure screen. held lists its entries as "KIND NAME",
// never a value: a field says whether its entry is set, and starts empty. A file entry holds bytes,
// and an env or git entry beyond these needs a name, so ws mac secrets sets those.
func secretsSheet(store secretsStore, held []string) tui.Sheet {
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
		Save: func(ctx context.Context, _ *tui.Bridge, edits map[string]string) (tui.Sheet, error) {
			var keys []string
			err := store.edit(ctx, func(s *secrets.Secrets) error {
				for _, label := range slices.Sorted(mapKeys(edits)) {
					if err := setEntry(s, label, edits[label]); err != nil {
						return err
					}
				}
				keys = s.Keys()
				return nil
			})
			if err != nil {
				return secretsSheet(store, held), err
			}
			return secretsSheet(store, keys), nil
		},
	}
}

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

func mapKeys(m map[string]string) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
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
