package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/keychain"
	"github.com/zadykian/ws/internal/secrets"
)

// secretsKey keeps the identity of a secrets file: the Mac's Keychain, or ws's credential.
type secretsKey interface {
	Identity(ctx context.Context) (age.Identity, error)
}

// secretsStore is the secrets file and the identity that opens it.
type secretsStore struct {
	m    effect.Machine
	k    secretsKey
	path string
	// write replaces the file with s, sealed to id.
	write func(s secrets.Secrets, id age.Identity) error
}

// openStore returns the store that a command edits, or why the command doesn't run.
type openStore func() (secretsStore, error)

// secretsEditCmds returns set, unset and keys, which edit the store that open returns.
func secretsEditCmds(p process, open openStore) []*cobra.Command {
	return []*cobra.Command{newSecretsSetCmd(p, open), {
		Use:   "unset KIND NAME",
		Short: "drop an entry",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			return editIn(c, open, func(s *secrets.Secrets) error { return s.Unset(args[0], args[1]) })
		},
	}, {
		Use:   "keys",
		Short: "print each entry as KIND NAME, never a value",
		Args:  cobra.NoArgs,
		RunE:  func(c *cobra.Command, _ []string) error { return secretsKeys(c, open) },
	}}
}

// editIn opens the store and edits it with fn.
func editIn(c *cobra.Command, open openStore, fn func(*secrets.Secrets) error) error {
	s, err := open()
	if err != nil {
		return err
	}
	return s.edit(c.Context(), fn)
}

// edit reads the file with its references as written, applies fn and writes the file again. A file
// that is missing is an empty one.
func (s secretsStore) edit(ctx context.Context, fn func(*secrets.Secrets) error) error {
	id, err := s.k.Identity(ctx)
	if err != nil {
		return err
	}
	cur, err := secrets.LoadRaw(s.m, s.path, id)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := fn(&cur); err != nil {
		return err
	}
	return s.write(cur, id)
}

// newSecretsSetCmd returns ws mac secrets set, with a command per kind.
func newSecretsSetCmd(p process, open openStore) *cobra.Command {
	cmds := []*cobra.Command{newSecretsSetGitCmd(open), newSecretsSetFileCmd(p, open)}
	for _, kind := range []string{secrets.KindToken, secrets.KindEnv, secrets.KindPassword} {
		cmds = append(cmds, &cobra.Command{
			Use:   kind + " NAME",
			Short: "set the " + kind + " NAME, from a prompt with no echo or from stdin",
			Args:  cobra.ExactArgs(1),
			RunE: func(c *cobra.Command, args []string) error {
				value, err := readValue(c.InOrStdin(), c.ErrOrStderr(), kind+" "+args[0])
				if err != nil {
					return err
				}
				return editIn(c, open, func(s *secrets.Secrets) error { return s.Set(kind, args[0], value) })
			},
		})
	}
	return group("set", "set an entry, whose value never goes in an argument", cmds...)
}

// newSecretsSetGitCmd returns ws mac secrets set git.
func newSecretsSetGitCmd(open openStore) *cobra.Command {
	var g secrets.Git
	c := &cobra.Command{
		Use:   "git URL",
		Short: "set a git server's token, from a prompt with no echo or from stdin",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			var err error
			if g.Token, err = readValue(c.InOrStdin(), c.ErrOrStderr(), secrets.KindGit+" "+args[0]); err != nil {
				return err
			}
			return editIn(c, open, func(s *secrets.Secrets) error {
				// A flag left out keeps the entry's, so that a form sets a token alone.
				cur := s.Git[args[0]]
				if !c.Flags().Changed("username") {
					g.Username = cur.Username
				}
				if !c.Flags().Changed("proactive-auth") {
					g.ProactiveAuth = cur.ProactiveAuth
				}
				s.SetGit(args[0], g)
				return nil
			})
		},
	}
	c.Flags().StringVar(&g.Username, "username", "", "the user name that git sends with the token, else the entry's")
	c.Flags().BoolVar(&g.ProactiveAuth, "proactive-auth", false, "send the token before the server asks for it, "+
		"else as the entry does")
	return c
}

// newSecretsSetFileCmd returns set file.
func newSecretsSetFileCmd(p process, open openStore) *cobra.Command {
	return &cobra.Command{
		Use:   "file NAME [PATH]",
		Short: "set a profile file (vpn_p12, vpn_sswan) from the file at PATH, or from stdin",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(c *cobra.Command, args []string) error {
			data, err := readProfile(c, p, args)
			if err != nil {
				return err
			}
			return editIn(c, open, func(s *secrets.Secrets) error { return s.SetFile(args[0], data) })
		},
	}
}

// readProfile reads the file at args[1], else stdin whole, as bytes: a terminal is refused, as no
// prompt takes a .p12.
func readProfile(c *cobra.Command, p process, args []string) ([]byte, error) {
	if len(args) == 2 {
		return p.machine.ReadFile(args[1])
	}
	if f, ok := c.InOrStdin().(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		return nil, fmt.Errorf("give the file %s as PATH, or on stdin from a file", args[0])
	}
	data, err := io.ReadAll(c.InOrStdin())
	if err != nil {
		return nil, fmt.Errorf("read the file %s: %w", args[0], err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("the file %s on stdin is empty", args[0])
	}
	return data, nil
}

// readValue reads a secret: at a terminal, a line with no echo, asked on stderr; else all of stdin,
// less one trailing newline. The error names no value.
func readValue(stdin io.Reader, stderr io.Writer, what string) (string, error) {
	var value string
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		_, _ = io.WriteString(stderr, what+": ")
		line, err := term.ReadPassword(int(f.Fd()))
		_, _ = io.WriteString(stderr, "\n")
		if err != nil {
			return "", fmt.Errorf("read the value of %s: %w", what, err)
		}
		value = string(line)
	} else {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("read the value of %s: %w", what, err)
		}
		value = strings.TrimSuffix(string(data), "\n")
	}
	if value == "" {
		return "", fmt.Errorf("the value of %s is empty", what)
	}
	return value, nil
}

// secretsKeys prints each entry as KIND NAME, never a value.
func secretsKeys(c *cobra.Command, open openStore) error {
	s, err := open()
	if err != nil {
		return err
	}
	id, err := s.k.Identity(c.Context())
	if err != nil {
		return err
	}
	cur, err := secrets.LoadRaw(s.m, s.path, id)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	for _, k := range cur.Keys() {
		_, _ = io.WriteString(c.OutOrStdout(), k+"\n")
	}
	return nil
}

// newSecretsStore returns the Mac's store: its file, and its identity in the Keychain.
func newSecretsStore(p process) secretsStore {
	path := filepath.Join(p.getenv("HOME"), secretsFile)
	return secretsStore{m: p.machine, k: keychain.Keychain{M: p.machine, User: p.getenv("USER")}, path: path,
		write: func(s secrets.Secrets, id age.Identity) error { return secrets.WriteFile(p.machine, path, s, id) }}
}

// sayOut prints a line on stdout after the command's path.
func sayOut(c *cobra.Command, format string, a ...any) {
	_, _ = fmt.Fprintf(c.OutOrStdout(), "%s: "+format+"\n", append([]any{c.CommandPath()}, a...)...)
}
