package jetbrains

import (
	"errors"
	"testing"
	"time"
)

func TestWithLockExcludesAnotherHolder(t *testing.T) {
	dir := t.TempDir()
	held := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error)
	go func() {
		done <- withLock(dir, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	second := make(chan struct{})
	go func() {
		_ = withLock(dir, func() error { close(second); return nil })
	}()
	select {
	case <-second:
		t.Fatal("the second holder ran under the first's lock")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	select {
	case <-second:
	case <-time.After(5 * time.Second):
		t.Fatal("the second holder never ran")
	}
}

func TestWithLockReturnsTheError(t *testing.T) {
	want := errors.New("boom")
	if err := withLock(t.TempDir(), func() error { return want }); !errors.Is(err, want) {
		t.Errorf("got %v", err)
	}
	if err := withLock(t.TempDir()+"/missing", func() error { return nil }); err == nil {
		t.Error("a missing directory is no error")
	}
}
