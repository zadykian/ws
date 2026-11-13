// Package aptrepo sets up the apt repositories of third parties: each one's key and .sources file.
package aptrepo

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
)

//go:embed repository.sources.tmpl
var templates embed.FS

// keyring and sourcesDir are where apt reads a repository's key and its .sources file.
const (
	keyring    = "/etc/apt/keyrings"
	sourcesDir = "/etc/apt/sources.list.d"
)

// Repository is a third-party apt repository: a deb822 .sources file, with its key in
// /etc/apt/keyrings. The role removes both files of a repository it no longer needs. A key fetched
// again must hold a certificate at least, and each one's primary fingerprint must be pinned here. A
// renewed key keeps its fingerprint, and a key the vendor drops leaves the rest pinned. Another
// key, in place of them or beside them, is not pinned.
type Repository struct {
	Name, Key, URIs, Suites, Components string
	// Fingerprints are the primary fingerprints of the certificates the key file may hold.
	Fingerprints []string
	Needed       bool
}

// Repositories sets up the apt repositories: it removes the sources they replace and those no
// longer needed, then writes each needed repository's key and .sources file. Apt refuses two
// sources of one repository whose keys differ, so the old sources go before the new ones come. A
// run that stops between the two leaves a repository missing, never defined twice.
type Repositories struct {
	M effect.Machine
	// List gives the repositories by the release's codename, and Replaced the files they replace.
	List     func(release string) []Repository
	Replaced func(release string) []string
}

// Plan fetches each key again at every run, so that a renewed key replaces the old one: Charm's
// expires in 2027. A key goes in only where its repository pins each certificate in it: apt trusts
// every certificate in its file. A vendor's server can fail for a moment, so curl tries
// again.
func (r Repositories) Plan(ctx context.Context) ([]resource.Change, error) {
	release, err := r.release()
	if err != nil {
		return nil, err
	}
	var parts []resource.Resource
	for _, path := range r.Replaced(release) {
		parts = append(parts, file.Absent{M: r.M, Path: path})
	}
	repos := r.List(release)
	for _, repo := range repos {
		if !repo.Needed {
			parts = append(parts,
				file.Absent{M: r.M, Path: fmt.Sprintf("%s/%s.sources", sourcesDir, repo.Name)},
				file.Absent{M: r.M, Path: fmt.Sprintf("%s/%s.asc", keyring, repo.Name)})
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
		sources, err := resource.Render(templates, "repository.sources.tmpl", repo)
		if err != nil {
			return nil, err
		}
		parts = append(parts,
			// The diff is off: a key that changes shows no diff, as it would be base64 alone.
			file.File{M: r.M, Path: fmt.Sprintf("%s/%s.asc", keyring, repo.Name), Content: key, Mode: 0o644, Secret: true},
			file.File{M: r.M, Path: fmt.Sprintf("%s/%s.sources", sourcesDir, repo.Name), Content: sources, Mode: 0o644})
	}
	var changes []resource.Change
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
