package wizard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

// release downloads WS_HOOK's release by the Mac's gh login, whatever git's config names: the build
// for ws's architecture and the sums, REPO-linux-ARCH and REPO.sha256, as ws's own releases have.
// It checks the build's SHA-256 against the sums, and returns the build with that sum. A tag that
// has no such assets, or sums that differ, stops the stage. The files go in the run's lock, which
// the run removes.
func (h *HookWs) release(ctx context.Context, hook string) ([]byte, string, error) {
	arch, err := h.Files.Do(ctx, "arch")
	if err != nil {
		return nil, "", err
	}
	gh, _, err := h.Mac.Gh.ghPath(ctx)
	if err != nil {
		return nil, "", err
	}
	repo, name, tag := hookRepo(hook), hookName(hook), hookTag(hook)
	asset := name + "-linux-" + strings.TrimSpace(string(arch))
	dir := h.Host.Lock + "/hook-release"
	if _, err := h.M.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		if err := h.M.Mkdir(dir, 0o700); err != nil {
			return nil, "", err
		}
	}
	h.Out.Say("%s: downloading %s of %s with the Mac's gh", hookWsStage, tag, repo)
	args := []string{"GH_PROMPT_DISABLED=1", gh, "release", "download", tag, "-R", repo, "-D", dir,
		"--clobber", "-p", asset, "-p", name + ".sha256"}
	if err := h.M.Pass(ctx, false, "env", args...); err != nil {
		return nil, "", hookStop{fmt.Errorf("%s: gh cannot download %s of %s, as above: the release needs %s and %s.sha256",
			hookWsStage, tag, repo, asset, name)}
	}
	data, err := h.M.ReadFile(dir + "/" + asset)
	if err != nil {
		return nil, "", err
	}
	sums, err := h.M.ReadFile(dir + "/" + name + ".sha256")
	if err != nil {
		return nil, "", err
	}
	got := sha256.Sum256(data)
	sum := hex.EncodeToString(got[:])
	if want := sumOf(sums, asset); want != sum {
		return nil, "", hookStop{fmt.Errorf("%s: the SHA-256 of %s in %s's release %s is not the one that %s.sha256 lists: "+
			"nothing was sent to ws", hookWsStage, asset, repo, tag, name)}
	}
	return data, sum, nil
}

// sumOf returns the SHA-256 that the sums, as sha256sum prints them, list for the file, in lower
// case, or an empty string where they list none.
func sumOf(sums []byte, file string) string {
	for _, l := range strings.Split(string(sums), "\n") {
		sum, name, ok := strings.Cut(strings.TrimSpace(l), " ")
		name = strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(name), "*"), "./")
		if ok && name == file {
			return strings.ToLower(sum)
		}
	}
	return ""
}

// upload sends the checked build to ws, to the path that the files script's stage names, and has the
// script install it, root's alone. The build goes by an ssh of its own, as the files script is on
// the first one's stdin.
func (h *HookWs) upload(ctx context.Context, hook string, data []byte, sum string) error {
	out, err := h.Files.Do(ctx, "stage", hook)
	if err != nil {
		return err
	}
	h.Out.Say("%s: sending %s to ws", hookWsStage, hookName(hook))
	path := strings.TrimSpace(string(out))
	if _, err := h.Host.SSH.RunInput(ctx, data, "-o", "BatchMode=yes", "ws-ssh", "umask 077 && cat >"+path); err != nil {
		return err
	}
	return h.Files.Pass(ctx, "install", hook, sum)
}
