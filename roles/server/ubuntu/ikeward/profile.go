package ikeward

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// The secret inputs of ikeward's mise.secrets.toml, which the secrets task's fnox sets in the run's
// env from the server's own profile. The .p12 is binary, so its input holds it in base64.
const (
	P12Var      = "WS_IKEWARD_P12"
	SswanVar    = "WS_IKEWARD_SSWAN"
	PasswordVar = "WS_IKEWARD_PASSWORD" //nolint:gosec // G101: a variable's name, no credential
)

// runDir is where the profile's files go for one run: in /run, where only root may enter. They keep
// their names, which name the files that install writes.
const runDir = "/run/ikeward-profile"

// Profile is the VPN's client profile, as the run's env gives it: the two files ikeward reads, and
// the .p12's password, empty where it has none.
type Profile struct {
	P12, Sswan []byte
	Password   string
}

// InRun reports whether the run's env, by lookup, sets either file's variable, as a run of the
// secrets task does. A run without them leaves the VPN alone.
func InRun(lookup func(string) (string, bool)) bool {
	_, p12Set := lookup(P12Var)
	_, sswanSet := lookup(SswanVar)
	return p12Set || sswanSet
}

// ProfileFrom reads the profile from the run's env by lookup. set is InRun's answer. An error
// names the variable alone, never its value.
func ProfileFrom(lookup func(string) (string, bool)) (p Profile, set bool, err error) {
	if !InRun(lookup) {
		return Profile{}, false, nil
	}
	p12, _ := lookup(P12Var)
	sswan, _ := lookup(SswanVar)
	// base64 without -w0 wraps its lines, so the decode drops any space between them.
	p.P12, err = base64.StdEncoding.DecodeString(strings.Join(strings.Fields(p12), ""))
	if err != nil {
		return Profile{}, true, fmt.Errorf("%s is not the .p12 in base64: %w", P12Var, err)
	}
	p.Sswan = []byte(sswan)
	p.Password, _ = lookup(PasswordVar)
	return p, true, nil
}

// profile is the profile's files, and the names they are written under.
type profile struct {
	m     effect.Machine
	name  string // the profile's path less the extension
	files Profile
}

// present reports whether the run gives both files.
func (p profile) present() bool { return len(p.files.P12) > 0 && len(p.files.Sswan) > 0 }

// base is the path of the written files, less the extension.
func (p profile) base() string { return path.Join(runDir, path.Base(p.name)) }

// with writes the files, with no output of their names or bytes, and runs f. It removes them
// whether f fails or not, even those a run killed before left in the directory.
func (p profile) with(ctx context.Context, f func() error) (err error) {
	defer func() {
		_, rmErr := p.m.RunRoot(ctx, "rm", "-rf", "--", runDir)
		err = errors.Join(err, rmErr)
	}()
	if err := p.m.Mkdir(runDir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	if err := p.m.Chmod(runDir, 0o700); err != nil {
		return err
	}
	for ext, data := range map[string][]byte{".p12": p.files.P12, ".sswan": p.files.Sswan} {
		if err := p.m.WriteRootFile(p.base()+ext, data, 0o600); err != nil {
			return err
		}
	}
	return f()
}
