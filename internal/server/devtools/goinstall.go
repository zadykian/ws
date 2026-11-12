package devtools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
)

// goBin is the go command of devtools' Go.
const goBin = "/usr/local/go/bin/go"

// goInstall builds a Go program as the user and puts it in the directory of path. The environment
// is set, not inherited: GOBIN puts the program where its path says, whatever GOPATH or GOBIN the
// user's go env sets. GOTOOLCHAIN=local builds it with this Go, never one go would download. GOENV=off
// ignores a go env -w file, which could turn off GOSUMDB, so go install checks the module against
// Go's checksum database.
func goInstall(ctx context.Context, m effect.Machine, user host.User, name, binary, pkg string) error {
	if _, err := m.Stat(goBin); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s is built with the Go in /usr/local/go, which the devtools role installs", name)
	} else if err != nil {
		return err
	}
	return user.Pass(ctx, m, "env", "GOBIN="+path.Dir(binary), "GOTOOLCHAIN=local", "GOENV=off",
		"GOSUMDB=sum.golang.org", goBin, "install", pkg)
}
