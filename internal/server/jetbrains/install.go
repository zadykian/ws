package jetbrains

import (
	"context"
	"path"
	"strconv"
)

// marker is what Gateway writes last, once an archive is unpacked, and so does the role. A
// directory that lacks it was cut short, and is installed again.
const marker = ".expandSucceeded"

// install puts one build in its directory: the archive is downloaded, checked against the SHA-256
// JetBrains publishes, and unpacked beside the directory, which then moves into place whole.
type install struct {
	s   Site
	b   Build
	sum string
}

func (c install) archive() string { return c.s.Downloads + "/" + c.b.Name + ".tar.gz" }

// unpack extracts the archive into DIR.part, whose contents the archive holds under one directory,
// as Gateway puts them in its own. The tree takes the build's name once whole and marked, so that
// a build cut short lacks it. mv replaces no directory Gateway has made there since, even an
// empty one.
func (c install) unpack(ctx context.Context) error {
	if _, err := c.s.M.Stat(c.b.Dir + "/" + marker); err == nil {
		return nil
	}
	part := c.b.Dir + ".part"
	if _, err := c.s.M.Run(ctx, "rm", "-rf", "--", part); err != nil {
		return err
	}
	if err := c.s.M.Mkdir(part, 0o755); err != nil {
		return err
	}
	if err := c.s.Unpack(c.archive(), part, 1); err != nil {
		return err
	}
	if err := c.s.M.WriteFile(path.Join(part, marker), nil, 0o644); err != nil {
		return err
	}
	owner := strconv.Itoa(c.s.UID) + ":" + strconv.Itoa(c.s.GID)
	if _, err := c.s.M.RunRoot(ctx, "chown", "-R", "-h", owner, "--", part); err != nil {
		return err
	}
	_, err := c.s.M.Run(ctx, "mv", "-T", "--update=none-fail", "--", part, c.b.Dir)
	return err
}
