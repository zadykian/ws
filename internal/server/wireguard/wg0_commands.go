package wireguard

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
)

type deleteLink struct{ m effect.Machine }

func (deleteLink) String() string { return "delete wg0, so that networkd creates it with the key" }

func (c deleteLink) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, "ip", "link", "delete", "wg0")
	return err
}

// Forced makes a failed run still apply the change, as a handler does.
func (deleteLink) Forced() bool { return true }

type reloadNetworkd struct {
	m   effect.Machine
	why string
}

func (c reloadNetworkd) String() string { return "reload networkd for wg0: " + c.why }

func (c reloadNetworkd) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, "networkctl", "reload")
	return err
}

// Forced makes a failed run still reload, as a handler does.
func (reloadNetworkd) Forced() bool { return true }
