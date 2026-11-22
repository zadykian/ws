package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/engine/core/effect"
)

// File is the one ws of a machine, which install.sh and ws update write.
const File = "/usr/local/bin/ws"

// Result is what an update did: File went From its version To a release. From is empty where File
// was missing or did not run, and is To where File was that release already, and nothing changed.
type Result struct{ From, To, File string }

// Report is what ws update says of r.
func (r Result) Report() string {
	switch r.From {
	case r.To:
		return fmt.Sprintf("%s is ws %s already\n", r.File, r.To)
	case "":
		return fmt.Sprintf("Installed ws %s: %s\n", r.To, r.File)
	}
	return fmt.Sprintf("Updated ws %s to %s: %s\n", r.From, r.To, r.File)
}

// Updater is one run of ws update.
type Updater struct {
	M effect.Machine
	// URL is the address of the releases, File the ws to bring to a release, and Binary the name
	// that the releases give ws for this machine.
	URL, File, Binary string
	// Tag is the release to install, vX.Y.Z or vX.Y.Z-rc.N. Empty takes the latest release.
	Tag string
	// PID names the copy that is checked beside File, so that two runs keep apart.
	PID int
}

// Run brings File to the release that Tag names, or the latest one. File's own version counts, as
// the units run File, whichever ws runs the update. A File of that release stays. A File of no
// release, a trial's build, or one that is missing or does not run, counts as older than any, and
// gets replaced. Without a Tag, a File of a release after the latest is refused, and a Tag names
// the older release to install.
func (u Updater) Run(ctx context.Context) (Result, error) {
	if !check.ReleasesURL(u.URL) {
		return Result{}, fmt.Errorf("WS_RELEASES_URL is %q: it must start with https://, or with "+
			"http://127.0.0.1: and hold no blank or quote", u.URL)
	}
	tag := u.Tag
	if tag != "" && !check.Release(tag) {
		return Result{}, fmt.Errorf("%q is no release's tag", tag)
	}
	from := u.version(ctx)
	if tag == "" {
		var err error
		if tag, err = latest(ctx, u.URL); err != nil {
			return Result{}, err
		}
		if check.Release(from) && semver.Compare(from, tag) > 0 {
			return Result{}, fmt.Errorf("%s is %s, later than the latest release, %s: "+
				"name the release to install by --version", u.File, from, tag)
		}
	}
	r := Result{From: from, To: tag, File: u.File}
	if from == tag {
		return r, nil
	}
	if err := u.replace(ctx, tag); err != nil {
		return Result{}, err
	}
	return r, nil
}

// version returns the version that File prints, or "" where File is missing or does not run.
func (u Updater) version(ctx context.Context) string {
	out, err := u.M.Run(ctx, u.File, "version")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// replace renames release tag's binary over File, once it matches its checksum and prints tag. The
// download is whole before it reaches the disk, and each write is a rename, so that a failure, or
// a cancelled ctx, leaves File unchanged. A ws that runs keeps its old file.
func (u Updater) replace(ctx context.Context, tag string) error {
	binary, err := u.download(ctx, u.URL+"/download/"+tag+"/")
	if err != nil {
		return err
	}
	// Root's, as launchd runs the file on a Mac only where root owns it, as ws unit does here.
	attrs := effect.Attrs{UID: 0, GID: 0, Mode: 0o755}
	// Beside File, as that directory runs ws: a temporary directory may be mounted noexec.
	copied := filepath.Join(filepath.Dir(u.File), fmt.Sprintf(".ws.%d", u.PID))
	if err := u.M.WriteAtomic(copied, binary, attrs); err != nil {
		return fmt.Errorf("writing beside %s: %w", u.File, err)
	}
	moved := false
	// Not ctx, which a signal may have cancelled.
	defer func() {
		if !moved {
			_, _ = u.M.RunRoot(context.WithoutCancel(ctx), "rm", "-f", copied)
		}
	}()
	out, err := u.M.Run(ctx, copied, "version")
	if err != nil {
		return fmt.Errorf("the %s downloaded does not run: %w", u.Binary, err)
	}
	if got := strings.TrimSpace(string(out)); got != tag {
		return fmt.Errorf("the %s downloaded is version %q, not %s", u.Binary, got, tag)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// mv would move the copy into a directory, where rename(2) fails.
	if info, err := u.M.Stat(u.File); err == nil && info.Mode.IsDir() {
		return fmt.Errorf("%s is a directory, not ws", u.File)
	}
	// The copy that printed tag is the file installed, with its owner and mode: no second write
	// shares a pending file with another run. Not ctx, so that a signal leaves the rename whole.
	if _, err := u.M.RunRoot(context.WithoutCancel(ctx), "mv", "-f", "--", copied, u.File); err != nil {
		return fmt.Errorf("replacing %s: %w", u.File, err)
	}
	moved = true
	return nil
}

// download returns the binary of the release at files, once it matches that release's ws.sha256.
func (u Updater) download(ctx context.Context, files string) ([]byte, error) {
	sums, err := get(ctx, files+"ws.sha256")
	if err != nil {
		return nil, err
	}
	want, ok := checksum(string(sums), u.Binary)
	if !ok {
		return nil, fmt.Errorf("%sws.sha256 has no checksum for %s", files, u.Binary)
	}
	binary, err := get(ctx, files+u.Binary)
	if err != nil {
		return nil, err
	}
	if sum := sha256.Sum256(binary); hex.EncodeToString(sum[:]) != want {
		return nil, fmt.Errorf("%s does not match its checksum in %sws.sha256", u.Binary, files)
	}
	return binary, nil
}
