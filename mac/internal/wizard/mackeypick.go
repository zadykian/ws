package wizard

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"
)

// other returns the .pub in Secretive's PublicKeys other than skip, as Secretive writes one for each
// key, or nothing. It fails where several are there.
func (k *MacKey) other(skip string) (string, error) {
	dir := k.secretive() + "/PublicKeys"
	names, err := k.M.ReadDir(dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	got := ""
	for _, name := range names {
		full := dir + "/" + name
		if !strings.HasSuffix(name, ".pub") || strings.HasSuffix(name, "-cert.pub") || strings.HasPrefix(name, ".") ||
			full == skip || !k.isFile(full) {
			continue
		}
		if got != "" {
			return "", fmt.Errorf("%s: Secretive holds several keys, %s and %s: name ws's in mac.conf's WS_IDENTITY, "+
				"and git's in WS_GIT_SIGNING_KEY", macKey, got, full)
		}
		got = full
	}
	return got, nil
}

func (k *MacKey) isFile(path string) bool {
	info, err := k.M.Stat(path)
	return err == nil && info.Mode.IsRegular()
}

// pick returns the .pub in PublicKeys other than skip. Where none is there, it says to make what in
// Secretive's app, adds extra, and waits up to 15 minutes for one. It fails on a key other than
// P-256, as an ML-DSA one, which ws's OpenSSH lacks.
func (k *MacKey) pick(ctx context.Context, skip, what, extra string) (string, error) {
	got, err := k.other(skip)
	if err == nil && got == "" {
		if extra != "" {
			extra = ". " + extra
		}
		k.Out.Say("%s: in Secretive, make %s; the wizard waits up to 15 minutes for it%s", macKey, what, extra)
		for left := 450; got == "" && err == nil; got, err = k.other(skip) {
			// A signal ends the wait at once, as wizard.sh's trap did.
			if err := ctx.Err(); err != nil {
				return "", err
			}
			if left--; left < 0 {
				return "", fmt.Errorf("%s: Secretive made no key in 15 minutes: make %s, then run the wizard again%s",
					macKey, what, extra)
			}
			k.Sleep(2 * time.Second)
		}
	}
	if err != nil {
		return "", err
	}
	text, _ := k.M.ReadFile(got)
	if !strings.HasPrefix(firstFields(string(text)), "ecdsa-sha2-nistp256 ") {
		return "", fmt.Errorf("%s: %s is not the P-256 key asked for: remove it in Secretive, make %s, then run the "+
			"wizard again", macKey, got, what)
	}
	return got, nil
}
