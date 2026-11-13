package cli

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// SessionOpen finds the session to ws of a sheet or a read, through the screen: the Mac's over ssh,
// or ws's own.
type SessionOpen func(ctx context.Context, b *tui.Bridge) (*session.Session, error)

// HostVarsSheet is the host_vars form before its read, which open reaches ws for. The password that
// sudo asks stays with the session, so that a save asks none.
func HostVarsSheet(open SessionOpen) tui.Sheet {
	return tui.Sheet{
		Name:   "host_vars",
		Fields: func(map[string]string) []tui.FieldSpec { return nil },
		Open: func(ctx context.Context, b *tui.Bridge) (tui.Sheet, error) {
			ws, err := open(ctx, b)
			if err != nil {
				return tui.Sheet{}, err
			}
			form, err := values.Form{Ws: ws}.Load(ctx)
			if err != nil {
				return tui.Sheet{}, err
			}
			return varsSheet(form), nil
		},
	}
}

// varsSheet is ws's host_vars on the Configure screen, as form read it.
func varsSheet(form values.Form) tui.Sheet {
	return tui.Sheet{
		Name:   "host_vars",
		Fields: func(map[string]string) []tui.FieldSpec { return enginecli.FieldSpecs(form.Fields()) },
		Save: func(ctx context.Context, b *tui.Bridge, edits map[string]string) (tui.Sheet, error) {
			next, err := form.Save(ctx, b, edits)
			return varsSheet(next), err
		},
	}
}

// secretsLabels are the entries that the form always offers, besides those the file holds.
var secretsLabels = []string{
	secrets.Label(secrets.KindToken, "github"), secrets.Label(secrets.KindToken, "azure_devops"),
	secrets.Label(secrets.KindPassword, "vpn_p12"),
}

// SecretsSheet is the secrets file on the Configure screen. held lists its entries as "KIND NAME",
// never a value: a field says whether its entry is set, and starts empty. A file entry holds bytes,
// and an env or git entry beyond these needs a name, so ws client macos secrets sets those.
func SecretsSheet(side SecretsSide, held []string) tui.Sheet {
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
			return SecretsSheet(side, keys), errors.Join(err, kerr)
		},
	}
}

// SecretsSide is where the secrets sheet reads and sets the file's entries, by label: "KIND NAME".
type SecretsSide interface {
	Keys(ctx context.Context) ([]string, error)
	Set(ctx context.Context, label, value string) error
}

// WsSide is ws's secrets file, which ws server ubuntu secrets edits on ws over the wizard's ssh, a
// value on ssh's stdin. set git, with no flag, keeps the entry's user name.
type WsSide struct{ Ws *session.Session }

// Set sets the entry label to value on ws.
func (s WsSide) Set(ctx context.Context, label, value string) error {
	kind, name, err := secrets.CutLabel(label)
	if err != nil {
		return err
	}
	_, err = s.Ws.Secrets(ctx, []byte(value+"\n"), "set", kind, "--", name)
	return err
}

// Keys lists the labels of the entries on ws.
func (s WsSide) Keys(ctx context.Context) ([]string, error) {
	out, err := s.Ws.Secrets(ctx, nil, "keys")
	if err != nil {
		return nil, fmt.Errorf("the secrets on ws: %w", err)
	}
	return strings.FieldsFunc(string(out), func(r rune) bool { return r == '\n' }), nil
}
