package base

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/engine/core/host"
)

// swapChange is one change of the swap, a closure and its words for --check.
type swapChange struct {
	text string
	do   func(context.Context) error
}

func (c swapChange) String() string { return c.text }

func (c swapChange) Apply(ctx context.Context) error { return c.do(ctx) }

// Plan reads the machine and returns the changes of its phase, in the order they apply.
func (s swap) Plan(_ context.Context) ([]host.Change, error) {
	st, err := s.state()
	if err != nil {
		return nil, err
	}
	p, err := planSwap(st)
	if err != nil {
		return nil, err
	}
	var cs []host.Change
	add := func(on bool, text string, do func(context.Context) error) {
		if on {
			cs = append(cs, swapChange{text, do})
		}
	}
	root := func(name string, args ...string) func(context.Context) error {
		return func(ctx context.Context) error { _, err := s.M.RunRoot(ctx, name, args...); return err }
	}
	if s.active {
		if !st.Booted {
			s.Out.Say("swap: swapon and swapoff skipped, systemd is not running")
		}
		add(p.Swapon, swapFile+" would be swapped to", root("swapon", swapFile))
		add(p.StopInstaller, "stop swapping to "+installerImg, root("swapoff", installerImg))
		return cs, nil
	}
	add(p.StopFile, "stop swapping to "+swapFile, root("swapoff", swapFile))
	add(p.RemoveFile, "remove "+swapFile, func(context.Context) error { return s.Disk.Remove(swapFile) })
	add(p.Create, fmt.Sprintf("make %s, %d MiB", swapFile, st.WantMiB), func(ctx context.Context) error { return s.create(ctx, st.WantMiB) })
	add(p.Restrict, "make "+swapFile+" root's, mode 0600", func(context.Context) error { return s.Disk.Restrict(swapFile) })
	return cs, nil
}

// create allocates the file under another name. Once mkswap has made it a swap area, the file
// takes the name /swapfile, so that a run cut short leaves no file that swapon would refuse. A file
// that a killed run left goes first.
func (s swap) create(ctx context.Context, size int64) error {
	tmp := swapFile + ".new"
	if err := s.Disk.Remove(tmp); err != nil {
		return err
	}
	if err := s.Disk.Allocate(tmp, size); err != nil {
		return err
	}
	if _, err := s.M.RunRoot(ctx, "mkswap", tmp); err != nil {
		return errors.Join(err, s.Disk.Remove(tmp))
	}
	if err := s.Disk.Rename(tmp, swapFile); err != nil {
		return errors.Join(err, s.Disk.Remove(tmp))
	}
	return nil
}
