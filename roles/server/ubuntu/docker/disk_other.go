//go:build !linux

package docker

import "errors"

// StatfsSize fails: Docker's server runs on Linux alone.
func StatfsSize(string) (uint64, error) { return 0, errors.New("statfs: Linux alone") }
