package kind

import (
	"errors"
	"fmt"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/locale"
	"github.com/zadykian/ws/engine/core/resource/swap"
	"github.com/zadykian/ws/engine/core/resource/timezone"
)

// The states of the system's own settings. Each takes a hint, which ends the refusal of a value
// that only the plan can check, as against the locales glibc lists: how to give another. Each
// state's resource is several parts in one (resource.Joined).

// localeState is the locale Name, generated and the system's (locale.Locale).
type localeState struct {
	Name string `toml:"name"`
	Hint string `toml:"hint"`
}

func (s localeState) resource(e Env) (resource.Resource, error) {
	if s.Name == "" {
		return nil, errors.New("a locale state takes a name")
	}
	return locale.Locale{M: e.M, Name: s.Name, Hint: s.Hint}, nil
}

// timezoneState is the time zone Name (timezone.Zone).
type timezoneState struct {
	Name string `toml:"name"`
	Hint string `toml:"hint"`
}

func (s timezoneState) resource(e Env) (resource.Resource, error) {
	if err := timezone.Check(s.Name); err != nil {
		return nil, joinHint(err, s.Hint)
	}
	return timezone.Zone{M: e.M, Name: s.Name, Hint: s.Hint}, nil
}

// swapState is the swap file, of SizeMiB MiB, none at 0, and the RAM's size where unset
// (swap.File). Its hint ends the refusal of a size that the disk lacks room for.
type swapState struct {
	SizeMiB *int   `toml:"size_mib"`
	Hint    string `toml:"hint"`
}

func (s swapState) resource(e Env) (resource.Resource, error) {
	if s.SizeMiB != nil && *s.SizeMiB < 0 {
		return nil, joinHint(fmt.Errorf("size_mib %d is no size: it takes a whole number of MiB, or 0 for none", *s.SizeMiB), s.Hint)
	}
	return swap.File{M: e.M, SizeMiB: s.SizeMiB, Hint: s.Hint}, nil
}

// joinHint ends err with the hint, where the state gives one.
func joinHint(err error, hint string) error {
	if hint == "" {
		return err
	}
	return errors.New(err.Error() + ". " + hint)
}
