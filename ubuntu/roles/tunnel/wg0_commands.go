package tunnel

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
)

type deleteLink struct {
	m    effect.Machine
	name string
}

func (c deleteLink) String() string {
	return "delete " + c.name + ", so that networkd creates it with the key"
}

func (c deleteLink) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, "ip", "link", "delete", c.name)
	return err
}

// Forced makes a failed run still apply the change, as a handler does.
func (deleteLink) Forced() bool { return true }

type reloadNetworkd struct {
	m         effect.Machine
	name, why string
}

func (c reloadNetworkd) String() string { return "reload networkd for " + c.name + ": " + c.why }

func (c reloadNetworkd) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, "networkctl", "reload")
	return err
}

// Forced makes a failed run still reload, as a handler does.
func (reloadNetworkd) Forced() bool { return true }
