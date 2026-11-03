package jetbrains

import (
	"strings"
	"testing"
)

func TestRoomIsShort(t *testing.T) {
	_, _, c := fixture(399)
	err := c.room()
	if err == nil || !strings.Contains(err.Error(), "needs 4 times") {
		t.Fatalf("got %v", err)
	}
}

func TestRoomIsEnough(t *testing.T) {
	_, _, c := fixture(400)
	if err := c.room(); err != nil {
		t.Error(err)
	}
}
