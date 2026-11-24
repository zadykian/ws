package values

import (
	"strings"
	"testing"
)

func TestParseRefusesACacheValueDockerdCantRead(t *testing.T) {
	for in, key := range map[string]string{
		"WS_DOCKER_BUILD_CACHE_KEEP_DAYS: -1\n":           "WS_DOCKER_BUILD_CACHE_KEEP_DAYS",
		"WS_DOCKER_BUILD_CACHE_KEEP_DAYS: 100000\n":       "WS_DOCKER_BUILD_CACHE_KEEP_DAYS",
		"WS_DOCKER_BUILD_CACHE_RESERVED_SPACE: 50Gi\n":    "WS_DOCKER_BUILD_CACHE_RESERVED_SPACE",
		"WS_DOCKER_BUILD_CACHE_MAX_USED_SPACE: ten\n":     "WS_DOCKER_BUILD_CACHE_MAX_USED_SPACE",
		"WS_DOCKER_BUILD_CACHE_MIN_FREE_SPACE: 5 GBB\n":   "WS_DOCKER_BUILD_CACHE_MIN_FREE_SPACE",
		"WS_DOCKER_BUILD_CACHE_MAX_USED_SPACE: 1 bytes\n": "WS_DOCKER_BUILD_CACHE_MAX_USED_SPACE",
		"WS_DOCKER_BUILD_CACHE_RESERVED_SPACE: 10%\n":     "WS_DOCKER_BUILD_CACHE_RESERVED_SPACE",
	} {
		if _, err := Parse([]byte(in)); err == nil || !strings.Contains(err.Error(), key) {
			t.Errorf("Parse(%q) = %v, want an error that names %s", in, err, key)
		}
	}
}

func TestParseTakesTheCacheFormats(t *testing.T) {
	for _, in := range []string{
		"WS_DOCKER_BUILD_CACHE_RESERVED_SPACE: 50G\n",
		"WS_DOCKER_BUILD_CACHE_MAX_USED_SPACE: 50GB\n",
		"WS_DOCKER_BUILD_CACHE_MIN_FREE_SPACE: 50GiB\n",
		"WS_DOCKER_BUILD_CACHE_MIN_FREE_SPACE: 1.5 tb\n",
		"WS_DOCKER_BUILD_CACHE_RESERVED_SPACE: 12b\n",
		"WS_DOCKER_BUILD_CACHE_RESERVED_SPACE: \"\"\n",
		"WS_DOCKER_BUILD_CACHE_KEEP_DAYS: 0\n",
		"WS_DOCKER_BUILD_CACHE_KEEP_DAYS: 99999\n",
	} {
		if _, err := Parse([]byte(in)); err != nil {
			t.Errorf("Parse(%q) = %v, want it taken", in, err)
		}
	}
}
