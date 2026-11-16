package local_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/zadykian/ws/engine/core/effect/local"
)

const (
	raceReads = 20000
	marker    = "root-only-marker"
)

// race plays the account: swap runs in a goroutine for as long as the reads go on, and each read
// of path that returns the marker is a leak. It needs no root.
func race(t *testing.T, home, path string, swap func(stop *atomic.Bool)) {
	var stop atomic.Bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		swap(&stop)
	}()
	var m local.OSMachine
	leaks := 0
	for range raceReads {
		if got, _, err := m.ReadHomeFile(home, path); err == nil && strings.Contains(string(got), marker) {
			leaks++
		}
	}
	stop.Store(true)
	<-done
	if leaks > 0 {
		t.Errorf("%d of %d reads returned the file behind a link", leaks, raceReads)
	}
}

func raceTree(t *testing.T) (home, outside string) {
	home, outside = homeTree(t)
	if err := os.WriteFile(outside, []byte(marker), 0o600); err != nil {
		t.Fatal(err)
	}
	return home, outside
}

func TestOSMachineReadHomeFileRacesALinkAtTheFile(t *testing.T) {
	home, outside := raceTree(t)
	path := filepath.Join(home, "f")
	race(t, home, path, func(stop *atomic.Bool) {
		for !stop.Load() {
			_ = os.Remove(path)
			_ = os.Symlink(outside, path)
			_ = os.Remove(path)
			_ = syscall.Mkfifo(path, 0o600)
			_ = os.Remove(path)
			_ = os.WriteFile(path, []byte("one"), 0o600)
		}
	})
}

func TestOSMachineReadHomeFileRacesAFileSwappedWithALink(t *testing.T) {
	home, outside := raceTree(t)
	path, link := filepath.Join(home, "f"), filepath.Join(home, "l")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	race(t, home, path, func(stop *atomic.Bool) {
		for !stop.Load() {
			_ = unix.Renameat2(unix.AT_FDCWD, path, unix.AT_FDCWD, link, unix.RENAME_EXCHANGE)
		}
	})
}

func TestOSMachineReadHomeFileRacesADirectoryExchangedWithALink(t *testing.T) {
	home, outside := raceTree(t)
	dir, link := filepath.Join(home, "d"), filepath.Join(home, "l")
	if err := os.Symlink(filepath.Dir(outside), link); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "outside"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	race(t, home, filepath.Join(dir, "outside"), func(stop *atomic.Bool) {
		for !stop.Load() {
			_ = unix.Renameat2(unix.AT_FDCWD, dir, unix.AT_FDCWD, link, unix.RENAME_EXCHANGE)
		}
	})
}
