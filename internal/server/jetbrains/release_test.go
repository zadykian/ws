package jetbrains

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

const list = `{"GO":[{"version":"2026.2.3","downloads":{"linux":{
  "link":"https://download.jetbrains.com/go/goland-2026.2.3.tar.gz","size":1200,
  "checksumLink":"https://download.jetbrains.com/go/goland-2026.2.3.tar.gz.sha256"}}}],
 "RD":[{"version":"2026.2.1","downloads":{"linux":{
  "link":"https://download.jetbrains.com/rider/JetBrains.Rider-2026.2.1.tar.gz","size":2400,
  "checksumLink":"https://download.jetbrains.com/rider/JetBrains.Rider-2026.2.1.tar.gz.sha256"}}}]}`

func TestParseReleasesNamesTheDirectoryAsGatewayDoes(t *testing.T) {
	got, err := ParseReleases([]byte(list), []string{"GO", "RD"}, "linux", "/h/dist")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Code != "GO" || got[1].Code != "RD" {
		t.Fatalf("builds in order of the codes: %+v", got)
	}
	sum := sha256.Sum256([]byte("https://download.jetbrains.com/go/goland-2026.2.3.tar.gz"))
	if want := fmt.Sprintf("/h/dist/%x_goland-2026.2.3", sum)[:len("/h/dist/")+13] + "_goland-2026.2.3"; got[0].Dir != want {
		t.Errorf("Dir = %s, want %s", got[0].Dir, want)
	}
	if got[0].Name != "goland-2026.2.3" || got[0].Size != 1200 {
		t.Errorf("build = %+v", got[0])
	}
	if got[1].Name != "JetBrains.Rider-2026.2.1" {
		t.Errorf("Name = %s", got[1].Name)
	}
}

func TestParseReleasesPatternMatchesOtherVersionsOfTheProductAlone(t *testing.T) {
	got, err := ParseReleases([]byte(list), []string{"GO"}, "linux", "/d")
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]bool{
		"0123456789abc_goland-2026.1.9":       true,
		"0123456789abc_goland-2026.2.3":       true,
		"0123456789abc_rider-2026.2.1":        false,
		"0123456789abc_goland-eap":            false,
		"short_goland-2026.1.9":               false,
		"0123456789abc_goland-community-2026": false,
	} {
		if got[0].Pattern.MatchString(name) != want {
			t.Errorf("Pattern on %s = %v, want %v", name, !want, want)
		}
	}
}

func TestParseReleasesRefuses(t *testing.T) {
	for name, tc := range map[string]struct{ data, code, platform string }{
		"unknown code":     {list, "XX", "linux"},
		"other platform":   {list, "GO", "linuxARM64"},
		"not json":         {"nope", "GO", "linux"},
		"link not tar.gz":  {strings.ReplaceAll(list, "2026.2.3.tar.gz\"", "2026.2.3.zip\""), "GO", "linux"},
		"name not version": {strings.ReplaceAll(list, `"version":"2026.2.3"`, `"version":"2027.1"`), "GO", "linux"},
		"link not https":   {strings.ReplaceAll(list, "https://download.jetbrains.com/go/goland-2026.2.3.tar.gz\"", "http://x/goland-2026.2.3.tar.gz\""), "GO", "linux"},
	} {
		if _, err := ParseReleases([]byte(tc.data), []string{tc.code}, tc.platform, "/d"); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
