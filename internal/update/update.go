package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/check"
)

// Result is what an update did: ws went From its release To the latest one, replacing File. Where
// File is empty, ws stayed at From, which is To or newer.
type Result struct{ From, To, File string }

// Report is what ws update says of r.
func (r Result) Report() string {
	switch {
	case r.File != "":
		return fmt.Sprintf("Updated ws %s to %s: %s\n", r.From, r.To, r.File)
	case r.From == r.To:
		return fmt.Sprintf("ws %s is the latest release\n", r.From)
	}
	return fmt.Sprintf("ws %s is newer than the latest release, %s\n", r.From, r.To)
}

// Updater is one run of ws update.
type Updater struct {
	M effect.Machine
	// Version is this ws's version, URL the address of the releases, File the file ws runs from,
	// and Binary the name that the releases give ws for this machine.
	Version, URL, File, Binary string
	// PID names the copy that is checked beside File, so that two runs keep apart.
	PID int
}

// Run replaces File with the latest release, where this ws is older. A build of no release has no
// version to compare, and is refused.
func (u Updater) Run(ctx context.Context) (Result, error) {
	if !check.Release(u.Version) {
		return Result{}, fmt.Errorf("this ws is %s, not a release: build it from its clone, "+
			"or install a release with install.sh", u.Version)
	}
	if !check.ReleasesURL(u.URL) {
		return Result{}, fmt.Errorf("WS_RELEASES_URL is %q: it must start with https://, or with "+
			"http://127.0.0.1: and hold no blank or quote", u.URL)
	}
	tag, err := latest(ctx, u.URL)
	if err != nil {
		return Result{}, err
	}
	r := Result{From: u.Version, To: tag}
	if semver.Compare(u.Version, tag) >= 0 {
		return r, nil
	}
	if err := u.replace(ctx, tag); err != nil {
		return Result{}, err
	}
	r.File = u.File
	return r, nil
}

// replace writes release tag's binary over File, once it matches its checksum and prints tag. The
// download is whole before it reaches the disk, and each write is a rename, so that a failure, or
// a cancelled ctx, leaves File unchanged. A ws that runs keeps its old file.
func (u Updater) replace(ctx context.Context, tag string) error {
	binary, err := u.download(ctx, u.URL+"/download/"+tag+"/")
	if err != nil {
		return err
	}
	info, err := u.M.Stat(u.File)
	if err != nil {
		return fmt.Errorf("reading %s: %w", u.File, err)
	}
	// An owner of -1 keeps the user who runs ws, as a file that user writes has.
	attrs := effect.Attrs{UID: -1, GID: -1, Mode: info.Mode.Perm()}
	// Beside File, as that directory runs ws: a temporary directory may be mounted noexec.
	copied := filepath.Join(filepath.Dir(u.File), fmt.Sprintf(".ws.%d", u.PID))
	if err := u.M.WriteAtomic(copied, binary, attrs); err != nil {
		return fmt.Errorf("writing beside %s: %w", u.File, err)
	}
	// Not ctx, which a signal may have cancelled.
	defer func() { _, _ = u.M.Run(context.WithoutCancel(ctx), "rm", "-f", copied) }()
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
	if err := u.M.WriteAtomic(u.File, binary, attrs); err != nil {
		return fmt.Errorf("replacing %s: %w", u.File, err)
	}
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
