package values

// Zed is roles/zed's keys.
type Zed struct {
	// Nofile is the soft limit of open files for root's logins. Zed's server busy-loops past
	// 1024 (ubuntu/roles/zed/README.md).
	Nofile int `yaml:"WS_ZED_NOFILE"`
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
	// ForwardFrom are the tunnel addresses of the WireGuard peers that reach the VPN through
	// this machine.
	ForwardFrom []string `yaml:"WS_IKEWARD_FORWARD_FROM"`
}

// Cable is roles/cable's keys.
type Cable struct {
	// Port is the cable's port. Empty, the role sets up no cable.
	Port string `yaml:"WS_CABLE_PORT"`
	// Address is the server's address on the cable, with its prefix length.
	Address string `yaml:"WS_CABLE_ADDRESS"`
}

// Security is roles/security's keys.
type Security struct {
	// Outbound turns on the outbound table: internet and ikeward0 alone.
	Outbound bool `yaml:"WS_SECURITY_OUTBOUND"`
}
