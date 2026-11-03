package jetbrains

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Build is the latest build of one backend: the release JetBrains lists, and where it goes.
type Build struct {
	// Code is JetBrains' product code, such as RD or GO.
	Code string
	// Link is the archive's URL, ChecksumLink the file with its SHA-256, and Size the archive's
	// size in bytes.
	Link, ChecksumLink string
	Size               int64
	// Name is the archive's name without .tar.gz.
	Name string
	// Dir is the build's directory under Gateway's: HASH_NAME, where HASH is the first 13 hex
	// digits of the SHA-256 of the link, which is how Gateway names the directory of a download.
	Dir string
	// Pattern matches the directory name of every build of the product, whatever its version.
	Pattern *regexp.Regexp
}

// release is one entry of JetBrains' list: a version, and a download for each platform.
type release struct {
	Version   string `json:"version"`
	Downloads map[string]struct {
		Link         string `json:"link"`
		Size         int64  `json:"size"`
		ChecksumLink string `json:"checksumLink"`
	} `json:"downloads"`
}

var (
	linkForm     = regexp.MustCompile(`^https://.+[.]tar[.]gz$`)
	checksumForm = regexp.MustCompile(`^https://`)
)

// ParseReleases reads JetBrains' list, a JSON object of releases by product code, and returns the
// latest build of each code, in the order of codes. platform is the list's name for this
// machine's platform and dist Gateway's directory of backends. It refuses a code with no
// .tar.gz and no SHA-256 for the platform, or whose archive is not named for its version.
func ParseReleases(data []byte, codes []string, platform, dist string) ([]Build, error) {
	var list map[string][]release
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("JetBrains' list of releases: %w", err)
	}
	builds := make([]Build, 0, len(codes))
	for _, code := range codes {
		var latest release
		if rs := list[code]; len(rs) > 0 {
			latest = rs[0]
		}
		file := latest.Downloads[platform]
		name := strings.TrimSuffix(path.Base(file.Link), ".tar.gz")
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(file.Link)))[:13]
		b := Build{
			Code: code, Link: file.Link, ChecksumLink: file.ChecksumLink, Size: file.Size,
			Name: name, Dir: dist + "/" + hash + "_" + name,
			Pattern: regexp.MustCompile(`^[0-9a-f]{13}_` + regexp.QuoteMeta(product(name, latest.Version)) + `[0-9]`),
		}
		if !linkForm.MatchString(b.Link) || !checksumForm.MatchString(b.ChecksumLink) ||
			!b.Pattern.MatchString(path.Base(b.Dir)) {
			return nil, fmt.Errorf("JetBrains lists no %s .tar.gz, with its SHA-256 and named for its version, "+
				"for the latest release of %s: is it a product code, such as RD or GO?", platform, code)
		}
		builds = append(builds, b)
	}
	return builds, nil
}

// product is the archive's name up to its version, as in goland-, which follows the hash in the
// directory of every build of that product. A name that lacks the version stays whole.
func product(name, version string) string {
	re := regexp.MustCompile(`^(.*?-)` + regexp.QuoteMeta(version) + `(-.*)?$`)
	return re.ReplaceAllString(name, "$1")
}
