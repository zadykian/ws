package jetbrains

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/zadykian/ws/effect"
)

// Site is what the backends' changes share: the machine, and where the builds and the downloads go.
type Site struct {
	M   effect.Machine
	Out effect.Out
	// Dist is Gateway's directory of backends, and Downloads where the archives are downloaded
	// before they are unpacked: the role's own directory, on disk where /tmp may be in memory.
	Dist, Downloads string
	// Home is the user's home, which Dist lies under. UID and GID are the user's, the owner of
	// everything the role makes there.
	Home     string
	UID, GID int
	// Prune removes the other builds of each product it installs.
	Prune bool
	// Avail returns the bytes free to the user on the filesystem that holds a path.
	Avail func(path string) (uint64, error)
	// Unpack extracts an archive into a directory, dropping the first strip components of names.
	Unpack func(archive, dir string, strip int) error
	// Wait is the pause between the five tries of a request or a download.
	Wait time.Duration
}

// tries is how often a request or a download is tried, as a server can fail for a moment.
const tries = 5

// retry runs fn up to tries times, waiting between them, and returns the last error.
func (s Site) retry(ctx context.Context, fn func() error) error {
	var err error
	for i := range tries {
		if err = fn(); err == nil {
			return nil
		}
		if i == tries-1 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.Wait):
		}
	}
	return err
}

// get reads a URL.
func (s Site) get(ctx context.Context, url string) ([]byte, error) {
	var body []byte
	err := s.retry(ctx, func() (err error) {
		body, err = s.M.Run(ctx, "curl", "-fsSL", url)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	return body, nil
}

// running returns the processes whose command line names the directory, as a backend's does.
// pgrep exits 1 where none matches.
func (s Site) running(ctx context.Context, dir string) ([]string, error) {
	out, err := s.M.Run(ctx, "pgrep", "-f", "--", dir+"/")
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(out)), nil
}
