package values

import (
	"net/netip"

	"github.com/zadykian/ws/internal/cablenet"
)

// Zed is roles/zed's keys.
type Zed struct {
	// Nofile is the soft limit of open files for root's logins. Zed's server busy-loops past
	// 1024 (ubuntu/roles/zed/README.md).
	Nofile int `yaml:"WS_ZED_NOFILE"`
}

// Nvidia is roles/nvidia's keys.
type Nvidia struct {
	// GPU is a guest's switch for the host's NVIDIA card. The host reads it from the guest's own
	// file and gives the guest the card. In a container, the role then installs the userspace
	// alone. A host's own file takes no such key.
	GPU bool `yaml:"WS_NVIDIA_GPU"`
}

// Ikeward is roles/ikeward's keys.
type Ikeward struct {
	// Profile is the IKEv2 profile's .p12 file. Empty, the role does nothing.
	Profile string `yaml:"WS_IKEWARD_PROFILE"`
	// DNS is the remote network's resolver, and Domains the domains whose names go to it alone.
	DNS     string   `yaml:"WS_IKEWARD_DNS"`
	Domains []string `yaml:"WS_IKEWARD_DOMAINS"`
	// Floors are the ranges kept off the uplink from boot on.
	Floors []string `yaml:"WS_IKEWARD_FLOORS"`
	// ForwardFrom are the tunnel addresses of the WireGuard peers. Any one turns forwarding on,
	// and every peer of wg0 then reaches the VPN.
	ForwardFrom []string `yaml:"WS_IKEWARD_FORWARD_FROM"`
}

// Cable is roles/cable's keys.
type Cable struct {
	// Port is the cable's port. Empty, the role sets up no cable.
	Port string `yaml:"WS_CABLE_PORT"`
}

// Address is the server's address on the cable, fixed with the Mac's end: 10.77.0.1/30.
func (Cable) Address() netip.Prefix {
	return cablenet.ServerPrefix()
}
