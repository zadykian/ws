package devtools

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// goReleases is go.dev's list of every release, with each file's SHA-256.
const goReleases = "https://go.dev/dl/?mode=json&include=all"

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// goSum returns the SHA-256 that the list gives the file, and fails where the list names no such
// file, or gives it no SHA-256.
func goSum(list []byte, filename string) (string, error) {
	var releases []struct {
		Files []struct {
			Filename string `json:"filename"`
			SHA256   string `json:"sha256"`
		} `json:"files"`
	}
	if err := json.Unmarshal(list, &releases); err != nil {
		return "", fmt.Errorf("go.dev's list of releases: %w", err)
	}
	for _, r := range releases {
		for _, f := range r.Files {
			if f.Filename == filename && sha256Hex.MatchString(f.SHA256) {
				return f.SHA256, nil
			}
		}
	}
	return "", fmt.Errorf("go.dev lists no %s with a SHA-256", filename)
}
