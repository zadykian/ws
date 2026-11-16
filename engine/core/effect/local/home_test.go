package local_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/local"
)

// homeTree makes a home with a file, a directory with a file, and a file outside the home that a
// link may point to.
func homeTree(t *testing.T) (home, outside string) {
	root := t.TempDir()
	home, outside = filepath.Join(root, "home"), filepath.Join(root, "outside")
	for path, text := range map[string]string{
		filepath.Join(home, "f"):      "one",
		filepath.Join(home, "d", "g"): "two",
		outside:                       "secret",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return home, outside
}

func TestOSMachineReadHomeFileReadsAFileAndItsAttributes(t *testing.T) {
	home, _ := homeTree(t)
	var m local.OSMachine
	for path, want := range map[string]string{filepath.Join(home, "f"): "one", filepath.Join(home, "d", "g"): "two"} {
		got, a, err := m.ReadHomeFile(home, path)
		if err != nil || string(got) != want {
			t.Errorf("ReadHomeFile(%s) = %q, %v; want %q", path, got, err, want)
		}
		if a.UID != os.Getuid() || a.GID != os.Getgid() || a.Mode != 0o600 {
			t.Errorf("ReadHomeFile(%s) attrs = %+v; want the file's owner and 0600", path, a)
		}
	}
}

func TestOSMachineReadHomeFileRefusesALinkAtTheFile(t *testing.T) {
	home, outside := homeTree(t)
	var m local.OSMachine
	for name, target := range map[string]string{"abs": outside, "rel": "f"} {
		link := filepath.Join(home, name)
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if got, _, err := m.ReadHomeFile(home, link); !errors.Is(err, syscall.ELOOP) {
			t.Errorf("a link to %s: %q, %v; want ELOOP", name, got, err)
		}
	}
}

func TestOSMachineReadHomeFileRefusesALinkOnTheWay(t *testing.T) {
	home, outside := homeTree(t)
	var m local.OSMachine
	for name, target := range map[string]string{"out": filepath.Dir(outside), "in": "d"} {
		if err := os.Symlink(target, filepath.Join(home, name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{filepath.Join(home, "out", "outside"), filepath.Join(home, "in", "g")} {
		if got, _, err := m.ReadHomeFile(home, path); !errors.Is(err, syscall.ENOTDIR) {
			t.Errorf("ReadHomeFile(%s) = %q, %v; want ENOTDIR", path, got, err)
		}
	}
}

func TestOSMachineReadHomeFileRefusesAFIFOWithoutWaiting(t *testing.T) {
	home, _ := homeTree(t)
	var m local.OSMachine
	if err := syscall.Mkfifo(filepath.Join(home, "p"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(home, "d", "q"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(home, "d", "q"), filepath.Join(home, "q")); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]error{
		filepath.Join(home, "p"):      effect.ErrNotRegular,
		filepath.Join(home, "q", "g"): syscall.ENOTDIR,
	} {
		done := make(chan error, 1)
		go func() {
			_, _, err := m.ReadHomeFile(home, path)
			done <- err
		}()
		select {
		case err := <-done:
			if !errors.Is(err, want) {
				t.Errorf("ReadHomeFile(%s): err = %v, want %v", path, err, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("ReadHomeFile(%s) waits on a FIFO", path)
		}
	}
}

func TestOSMachineReadHomeFileRefusesADirectory(t *testing.T) {
	home, _ := homeTree(t)
	var m local.OSMachine
	if _, _, err := m.ReadHomeFile(home, filepath.Join(home, "d")); !errors.Is(err, effect.ErrNotRegular) {
		t.Errorf("a directory: err = %v, want ErrNotRegular", err)
	}
}

func TestOSMachineReadHomeFileRefusesAFileOverTheMax(t *testing.T) {
	home, _ := homeTree(t)
	var m local.OSMachine
	path := filepath.Join(home, "f")
	if err := os.Truncate(path, effect.HomeFileMax); err != nil {
		t.Fatal(err)
	}
	if got, _, err := m.ReadHomeFile(home, path); err != nil || len(got) != effect.HomeFileMax {
		t.Errorf("a file of the max: %d bytes, %v; want read whole", len(got), err)
	}
	if err := os.Truncate(path, effect.HomeFileMax+1); err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.ReadHomeFile(home, path); !errors.Is(err, effect.ErrTooLarge) {
		t.Errorf("a file over the max: err = %v, want ErrTooLarge", err)
	}
}

func TestOSMachineReadHomeFileGivesNotExistForAMissingFile(t *testing.T) {
	home, _ := homeTree(t)
	var m local.OSMachine
	for _, path := range []string{filepath.Join(home, "none"), filepath.Join(home, "none", "g")} {
		if _, _, err := m.ReadHomeFile(home, path); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("ReadHomeFile(%s): err = %v, want fs.ErrNotExist", path, err)
		}
	}
}

func TestOSMachineReadHomeFileRefusesAPathNotBelowTheHome(t *testing.T) {
	home, outside := homeTree(t)
	var m local.OSMachine
	for _, path := range []string{outside, home, filepath.Join(home, "..", "outside")} {
		if got, _, err := m.ReadHomeFile(home, path); err == nil {
			t.Errorf("ReadHomeFile(%s) = %q; want an error", path, got)
		}
	}
}
