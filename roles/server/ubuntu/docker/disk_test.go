package docker

import (
	"errors"
	"testing"
)

func TestDataDiskSizeReadsDockersRootOrItsParent(t *testing.T) {
	var asked []string
	size := func(path string) (uint64, error) { asked = append(asked, path); return 42, nil }
	for _, c := range []struct {
		exists bool
		want   string
	}{{true, "/var/lib/docker"}, {false, "/var/lib"}} {
		asked = nil
		got, err := dataDiskSize(func(string) bool { return c.exists }, size)
		if err != nil || got != 42 || len(asked) != 1 || asked[0] != c.want {
			t.Errorf("exists %v: got %d, %v after asking %q; want 42 after %q", c.exists, got, err, asked, c.want)
		}
	}
}

func TestDataDiskSizeNamesThePathItFailedOn(t *testing.T) {
	_, err := dataDiskSize(func(string) bool { return false },
		func(string) (uint64, error) { return 0, errors.New("boom") })
	if err == nil {
		t.Error("a failed read passed")
	}
}
