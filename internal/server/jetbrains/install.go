package jetbrains

import "context"

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
// a build cut short lacks it. All of it runs as the user, tar too, which refuses names with "..". mv replaces no directory Gateway has made there since, even an
// empty one.
func (c install) unpack(ctx context.Context) error {
	if _, err := c.s.M.Stat(c.b.Dir + "/" + marker); err == nil {
		return nil
	}
	part := c.b.Dir + ".part"
	steps := [][]string{
		{"rm", "-rf", "--", part},
		{"mkdir", "-m", "0755", "--", part},
		{"tar", "-xzf", c.archive(), "-C", part, "--strip-components=1"},
		{"touch", "--", part + "/" + marker},
		{"mv", "-T", "--update=none-fail", "--", part, c.b.Dir},
	}
	for _, argv := range steps {
		if _, err := c.s.AsUser(ctx, argv[0], argv[1:]...); err != nil {
			return err
		}
	}
	return nil
}
