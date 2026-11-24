package values

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestParseHostEmptyGivesDefault(t *testing.T) {
	got, err := ParseHost(nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := DefaultHost(); !hostAlike(t, got, want) {
		t.Errorf("ParseHost(nil) = %+v, want %+v", got, want)
	}
}

// hostAlike says whether a and b give yaml the same text.
func hostAlike(t *testing.T, a, b Host) bool {
	textA, errA := yaml.Marshal(a)
	textB, errB := yaml.Marshal(b)
	if errA != nil || errB != nil {
		t.Fatal(errA, errB)
	}
	return string(textA) == string(textB)
}

// A host's file holds the keys of its own roles, and no guest's: a key that no role of the host
// reads is a typo there.
func TestParseHostRefusesAGuestsKey(t *testing.T) {
	for _, key := range []string{"WS_GIT_SIGNING_PUB_KEY", "WS_TUNNEL_PORT", "WS_ZED_NOFILE", "WS_NVIDIA_GPU"} {
		if _, err := ParseHost([]byte(key + ": x\n")); err == nil || !strings.Contains(err.Error(), "field "+key) {
			t.Errorf("ParseHost with %s = %v, want it refused as no key of the host's", key, err)
		}
	}
}

func TestParseHostChecksTheSectionsItHolds(t *testing.T) {
	for in, want := range map[string]string{
		"WS_BASE_SWAPPINESS: 400\n":               "WS_BASE_SWAPPINESS is 400",
		"WS_DOCKER_BUILD_CACHE_KEEP_DAYS: -1\n":   "WS_DOCKER_BUILD_CACHE_KEEP_DAYS is -1",
		"WS_CABLE_PORT: a/b\n":                    "WS_CABLE_PORT",
		"WS_DOCKER_BUILD_CACHE_MAX_USED_SPACE: x": "WS_DOCKER_BUILD_CACHE_MAX_USED_SPACE",
	} {
		if _, err := ParseHost([]byte(in)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("ParseHost(%q) = %v, want it to say %q", in, err, want)
		}
	}
}
