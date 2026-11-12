package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/wizard"
)

// secretsLabels are the entries that the form always offers, besides those the file holds.
var secretsLabels = []string{
	secrets.Label(secrets.KindToken, "github"), secrets.Label(secrets.KindToken, "azure_devops"),
	secrets.Label(secrets.KindPassword, "vpn_p12"),
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
				if kind, _, err := secrets.CutLabel(k); err == nil && (kind == secrets.KindEnv || kind == secrets.KindGit) {
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
				if err = side.Set(ctx, label, edits[label]); err != nil {
					break
				}
			}
			keys, kerr := side.Keys(ctx)
			if kerr != nil {
				keys = held
			}
			return secretsSheet(side, keys), errors.Join(err, kerr)
		},
	}
}

// secretsSide is where the secrets sheet reads and sets the file's entries, by label: "KIND NAME".
type secretsSide interface {
	Keys(ctx context.Context) ([]string, error)
	Set(ctx context.Context, label, value string) error
}

// wsSide is ws's secrets file, which ws server secrets edits on ws over the wizard's ssh, a value on
// ssh's stdin. set git, with no flag, keeps the entry's user name.
type wsSide struct{ ws *wizard.Session }

func (s wsSide) Set(ctx context.Context, label, value string) error {
	kind, name, err := secrets.CutLabel(label)
	if err != nil {
		return err
	}
	_, err = s.ws.Secrets(ctx, []byte(value+"\n"), "set", kind, "--", name)
	return err
}

func (s wsSide) Keys(ctx context.Context) ([]string, error) {
	out, err := s.ws.Secrets(ctx, nil, "keys")
	if err != nil {
		return nil, fmt.Errorf("the secrets on ws: %w", err)
	}
	return strings.FieldsFunc(string(out), func(r rune) bool { return r == '\n' }), nil
}

// secretsSheet is ws's secrets sheet before its read, which reaches ws as the host_vars sheet does.
func (r *varsReach) secretsSheet() tui.Sheet {
	return tui.Sheet{
		Name:   "secrets",
		Fields: func(map[string]string) []tui.FieldSpec { return nil },
		Open: func(ctx context.Context, b *tui.Bridge) (tui.Sheet, error) {
			ws, err := r.reach(ctx, b)
			if err != nil {
				return tui.Sheet{}, err
			}
			side := wsSide{ws}
			held, err := side.Keys(ctx)
			if err != nil {
				return tui.Sheet{}, err
			}
			return secretsSheet(side, held), nil
		},
	}
}
