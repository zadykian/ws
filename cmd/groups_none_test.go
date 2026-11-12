//go:build !darwin && !linux

package main

import "testing"

func TestNoGroupsHere(t *testing.T) {
	if got := groups(fakeProcess()); len(got) != 0 {
		t.Errorf("groups = %d, want none", len(got))
	}
}
