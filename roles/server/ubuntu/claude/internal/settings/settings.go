// Package settings sets up claude's user settings and its telemetry in them.
package settings

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
)

// UserSettings has cld add to claude's user settings what they lack. It adds the permission set,
// cld's choices of model, theme and the like, and the channel claude notifies on, where one is
// given.
// cld keeps every value already there, and leaves env, which Telemetry writes, alone.
type UserSettings struct {
	M    effect.Machine
	User account.User
	// Permissions is read-only, cld or none. Notifications is the channel, or empty for none: cld
	// refuses an empty channel, so without one the option is left out.
	Permissions, Notifications string
}

func (s UserSettings) cld() string { return s.User.Home + "/.local/bin/cld" }

func (s UserSettings) dir() string { return s.User.Home + "/.claude" }

func (s UserSettings) args() []string {
	args := []string{"setup", "config", "user", "--permissions=" + s.Permissions}
	if s.Notifications != "" {
		args = append(args, "--notifications="+s.Notifications)
	}
	return args
}

type setUserSettings struct {
	s      UserSettings
	report string
}

// String shows cld's report: it names the keys it would change, not their values, as the file
// may hold tokens.
func (c setUserSettings) String() string {
	return "cld setup config user changes claude's user settings:\n" + c.report
}

// Apply runs cld on the real file. CLAUDE_CONFIG_DIR is set, so that cld edits the file that
// Telemetry edits, whatever the caller's is.
func (c setUserSettings) Apply(ctx context.Context) error {
	return c.s.User.Pass(ctx, c.s.M, "env", append([]string{"CLAUDE_CONFIG_DIR=" + c.s.dir(), c.s.cld()}, c.s.args()...)...)
}

// Plan runs cld on a copy of the file, removed after, and reads its report: Left where it would
// change nothing. Without cld, the install step reports the change.
func (s UserSettings) Plan(ctx context.Context) (_ []resource.Change, err error) {
	if _, err := s.M.Stat(s.cld()); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	settings := s.dir() + "/settings.json"
	info, err := s.M.Stat(settings)
	switch {
	case err != nil && !errors.Is(err, fs.ErrNotExist):
		return nil, err
	case err == nil && info.Link != "" && info.Mode == 0:
		// cld refuses a link to no file, so the plan fails where the run would.
		return nil, fmt.Errorf("%s is a symbolic link to a file that does not exist", settings)
	}
	out, err := s.User.Output(ctx, s.M, "mktemp", "-d")
	if err != nil {
		return nil, err
	}
	copyDir := strings.TrimSpace(string(out))
	defer func() {
		if _, rmErr := s.User.Output(ctx, s.M, "rm", "-rf", "--", copyDir); err == nil {
			err = rmErr
		}
	}()
	if info.Mode != 0 {
		if _, err := s.User.Output(ctx, s.M, "cp", "--", settings, copyDir+"/settings.json"); err != nil {
			return nil, err
		}
	}
	report, err := s.User.Output(ctx, s.M, "env", append([]string{"CLAUDE_CONFIG_DIR=" + copyDir, s.cld()}, s.args()...)...)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(strings.ReplaceAll(string(report), copyDir+"/settings.json", settings))
	if text == "" || strings.HasPrefix(text, "Left ") {
		return nil, nil
	}
	return []resource.Change{setUserSettings{s, text}}, nil
}
