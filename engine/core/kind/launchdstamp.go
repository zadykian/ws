package kind

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// stampDir holds a stamp a daemon, named by its label: the SHA-256 of each file it watches, as it
// was when the daemon last started or read its files. launchd hashes neither the plist's program
// nor the files it reads, so the stamp tells a change to them. Root's, and readable by all.
const stampDir = "/var/db/ws/launchd"

// missingSum stands for a path that is missing, so that its making counts as a change.
const missingSum = "-"

func stampPath(label string) string { return path.Join(stampDir, label) }

// sum is one line of a stamp: a watched path and its SHA-256, in hex.
type sum struct{ path, sum string }

// sums is a stamp: the program first, then restart_on's paths and reload_on's.
type sums []sum

// text is the stamp's file: a line a path, its sum, two spaces and the path, as sha256sum prints.
func (s sums) text() string {
	var b strings.Builder
	for _, l := range s {
		fmt.Fprintf(&b, "%s  %s\n", l.sum, l.path)
	}
	return b.String()
}

// parseStamp reads a stamp's file into each path's sum. A line of another form counts for none.
func parseStamp(data []byte) map[string]string {
	was := map[string]string{}
	for line := range strings.Lines(string(data)) {
		if s, p, ok := strings.Cut(strings.TrimSuffix(line, "\n"), "  "); ok {
			was[p] = s
		}
	}
	return was
}

// differs reports whether a path of paths has another sum now than the stamp kept, or none kept.
func differs(was map[string]string, now sums, paths []string) bool {
	return slices.ContainsFunc(now, func(l sum) bool { return slices.Contains(paths, l.path) && was[l.path] != l.sum })
}

// stamp sums the program and each watched path of the daemon, as they are now.
func (u daemonUnit) stamp() (sums, error) {
	var s sums
	for _, p := range slices.Concat([]string{u.program}, u.d.RestartOn, u.d.ReloadOn) {
		h, err := pathSum(u.e.M, p)
		if err != nil {
			return nil, err
		}
		s = append(s, sum{p, h})
	}
	return s, nil
}

// pathSum is a file's SHA-256, or a directory's as compose's sum of the files below it, in hex. A
// missing path is missingSum.
func pathSum(m effect.Machine, p string) (string, error) {
	info, err := m.Stat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return missingSum, nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode.IsDir() {
		return projectSum(m, p)
	}
	data, err := m.ReadRootFile(p)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// keepStamp writes the daemon's stamp, after the call of launchctl that the plan holds.
type keepStamp struct{ u daemonUnit }

func (c keepStamp) String() string {
	return "keep the sums of the files of " + c.u.label + " in " + stampPath(c.u.label)
}

func (keepStamp) Kind() step.ChangeKind { return step.Write }

// Apply sums the files again, as the daemon has just read them. A failed call before it leaves
// the old stamp, so that the next run calls again.
func (c keepStamp) Apply(ctx context.Context) error {
	now, err := c.u.stamp()
	if err != nil {
		return err
	}
	if _, err := c.u.e.M.RunRoot(ctx, "mkdir", "-p", "--", stampDir); err != nil {
		return err
	}
	return c.u.e.M.WriteRootFile(stampPath(c.u.label), []byte(now.text()), 0o644)
}
