package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/zadykian/ws/effect"
)

// Updater is one run of ws update.
type Updater struct {
	M effect.Machine
	// Version is this ws's version, URL the address of the releases, File the file ws runs from,
	// and Binary the name that the releases give ws for this machine.
	Version, URL, File, Binary string
	// PID names the copy that is checked beside File, so that two runs keep apart.
	PID int
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
