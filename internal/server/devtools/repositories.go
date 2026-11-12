package devtools

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
)

//go:embed repository.sources.tmpl
var templates embed.FS

// keyring and sourcesDir are where apt reads a repository's key and its .sources file.
const (
	keyring    = "/etc/apt/keyrings"
	sourcesDir = "/etc/apt/sources.list.d"
)

// Repositories sets up the apt repositories: it removes the sources they replace and those no
// longer needed, then writes each needed repository's key and .sources file. Apt refuses two
// sources of one repository whose keys differ, so the old sources go before the new ones come. A
// run that stops between the two leaves a repository missing, never defined twice.
type Repositories struct {
	M effect.Machine
	// repos lists the repositories by the release's codename. Nil means repositories; tests pin
	// their own keys.
	repos func(release string) []Repository
}

// Plan fetches each key again at every run, so that a renewed key replaces the old one: Charm's
// expires in 2027. A key goes in only where its repository pins each certificate in it: apt trusts
// every certificate in its file. A vendor's server can fail for a moment, so curl tries
// again.
func (r Repositories) Plan(ctx context.Context) ([]host.Change, error) {
	release, err := r.release()
	if err != nil {
		return nil, err
	}
	var parts []host.Resource
	for _, path := range replacedFiles(release) {
		parts = append(parts, host.Absent{M: r.M, Path: path})
	}
	list := repositories
	if r.repos != nil {
		list = r.repos
	}
	repos := list(release)
	for _, repo := range repos {
		if !repo.Needed {
			parts = append(parts,
				host.Absent{M: r.M, Path: fmt.Sprintf("%s/%s.sources", sourcesDir, repo.Name)},
				host.Absent{M: r.M, Path: fmt.Sprintf("%s/%s.asc", keyring, repo.Name)})
		}
	}
	for _, repo := range repos {
		if !repo.Needed {
			continue
		}
		key, err := fetchKey(ctx, r.M, repo)
		if err != nil {
			return nil, err
		}
		sources, err := host.Render(templates, "repository.sources.tmpl", repo)
		if err != nil {
			return nil, err
		}
		parts = append(parts,
			// The diff is off: a key that changes shows no diff, as it would be base64 alone.
			host.File{M: r.M, Path: fmt.Sprintf("%s/%s.asc", keyring, repo.Name), Content: key, Mode: 0o644, Secret: true},
			host.File{M: r.M, Path: fmt.Sprintf("%s/%s.sources", sourcesDir, repo.Name), Content: sources, Mode: 0o644})
	}
	var changes []host.Change
	for _, part := range parts {
		c, err := part.Plan(ctx)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c...)
	}
	return changes, nil
}

// release returns the Ubuntu release's codename, such as resolute.
func (r Repositories) release() (string, error) {
	data, err := r.M.ReadFile("/usr/lib/os-release")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(line, "VERSION_CODENAME="); ok {
			return strings.Trim(v, `"'`), nil
		}
	}
	return "", fmt.Errorf("/usr/lib/os-release names no VERSION_CODENAME")
}
