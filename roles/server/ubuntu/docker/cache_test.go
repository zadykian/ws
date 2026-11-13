package docker_test

import (
	"testing"

	"github.com/zadykian/ws/roles/server/ubuntu/docker"
)

func TestDefaultCacheIsASharePlusOneGiB(t *testing.T) {
	const gib = 1 << 30
	got := docker.DefaultCache(500 * gib)
	if want := (docker.Cache{Reserved: "51GB", MaxUsed: "401GB", MinFree: "101GB"}); got != want {
		t.Errorf("DefaultCache(500 GiB) = %+v, want %+v", got, want)
	}
}

func TestCacheCheck(t *testing.T) {
	ok := docker.Cache{Reserved: "50G", MaxUsed: "80GB", MinFree: "1.5 TiB"}
	for name, c := range map[string]struct {
		cache docker.Cache
		days  int
		ok    bool
	}{
		"sizes in each form":  {ok, 90, true},
		"no age limit":        {ok, 0, true},
		"the longest age":     {ok, 99999, true},
		"an age too long":     {ok, 100000, false},
		"a negative age":      {ok, -1, false},
		"Gi without b":        {docker.Cache{Reserved: "50Gi", MaxUsed: "1G", MinFree: "1G"}, 1, false},
		"a size with a unit":  {docker.Cache{Reserved: "1G", MaxUsed: "1 bytes", MinFree: "1G"}, 1, false},
		"an empty size":       {docker.Cache{Reserved: "1G", MaxUsed: "1G"}, 1, false},
		"a percent of a disk": {docker.Cache{Reserved: "10%", MaxUsed: "1G", MinFree: "1G"}, 1, false},
	} {
		if err := c.cache.Check(c.days); (err == nil) != c.ok {
			t.Errorf("%s: Check(%d) = %v, want ok %v", name, c.days, err, c.ok)
		}
	}
}
