//go:build !linux

package jetbrains

import "errors"

// StatfsAvail fails: the server runs on Linux alone.
func StatfsAvail(string) (uint64, error) { return 0, errors.New("statfs: Linux alone") }
