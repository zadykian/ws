package git

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/internal/gitcfg"
)

// shadow finds what in the account's files shadows ws's git settings. Creds are the tokens of
// TokensFile, whose URLs tokens.gitconfig gives a program.
type shadow struct {
	M     effect.Machine
	A     account.Account
	Creds []Credential
}

// notes returns the notes of ~/.gitconfig's text. Where the include is to be added, they name the
// file's settings of identity and signing, which win over /etc/ws/gitconfig's below the include.
// They name those of git's XDG file too, which lose to them. Each time, they name the credential
// sections of the file that reset a token's credential helper.
func (s shadow) notes(ctx context.Context, text []byte, adding bool) ([]string, error) {
	var notes []string
	if adding {
		names, err := gitNames(ctx, s.M, text)
		if err != nil {
			return nil, err
		}
		if own := gitcfg.Own(names); own != "" {
			notes = append(notes, fmt.Sprintf("~/.gitconfig's own settings of %s, below the include, win over %s's", own, gitcfg.Path))
		}
		xdg, err := s.xdg(ctx)
		if err != nil {
			return nil, err
		}
		notes = append(notes, xdg...)
	}
	creds, err := s.credentials(ctx, text)
	if err != nil {
		return nil, err
	}
	return append(notes, creds...), nil
}

// xdg returns the note of the settings of identity and signing in git's XDG file, which git reads
// before ~/.gitconfig, so that /etc/ws/gitconfig's win over them. It reads below the home alone,
// as the run has no XDG_CONFIG_HOME of the account's.
func (s shadow) xdg(ctx context.Context) ([]string, error) {
	rel, err := filepath.Rel(s.A.Home, gitcfg.XDGFile("", s.A.Home))
	if err != nil {
		return nil, err
	}
	text, err := s.A.ReadHome(s.M, rel)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	names, err := gitNames(ctx, s.M, text)
	if err != nil {
		return nil, err
	}
	if own := gitcfg.Own(names); own != "" {
		return []string{gitcfg.XDGNote(s.A.Home+"/"+rel, own, gitcfg.Path)}, nil
	}
	return nil, nil
}
