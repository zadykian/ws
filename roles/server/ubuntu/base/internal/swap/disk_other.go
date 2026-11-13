//go:build !linux

package swap

import "errors"

// osDisk is the disk of a machine that is no Linux: each call fails, as ws sets a server up from
// Linux alone.
type osDisk struct{}

var errLinuxOnly = errors.New("the swap file is set up on Linux alone")

func (osDisk) Statfs(string) (int64, error)      { return 0, errLinuxOnly }
func (osDisk) Stat(string) (swapFileInfo, error) { return swapFileInfo{}, errLinuxOnly }
func (osDisk) Allocate(string, int64) error      { return errLinuxOnly }
func (osDisk) Restrict(string) error             { return errLinuxOnly }
func (osDisk) Rename(string, string) error       { return errLinuxOnly }
func (osDisk) Remove(string) error               { return errLinuxOnly }
