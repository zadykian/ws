package values

// Helix is roles/helix's keys.
type Helix struct {
	Grammars []string `yaml:"helix_grammars"`
}

// Zed is roles/zed's keys.
type Zed struct {
	// Nofile is the soft limit of open files for ws_user's logins. Zed's server busy-loops past
	// 1024 (docs/zed.md).
	Nofile int `yaml:"zed_nofile"`
}

// Ikeward is roles/ikeward's keys.
type Ikeward struct {
	// Profile is the IKEv2 profile's .p12 file. Empty, the role does nothing.
	Profile string `yaml:"ikeward_profile"`
	// DNS is the remote network's resolver, and Domains the domains whose names go to it alone.
	DNS     string   `yaml:"ikeward_dns"`
	Domains []string `yaml:"ikeward_domains"`
	// Floors are the ranges kept off the uplink from boot on.
	Floors []string `yaml:"ikeward_floors"`
	// ForwardFrom are the tunnel addresses of the WireGuard peers that reach the VPN through
	// this machine.
	ForwardFrom []string `yaml:"ikeward_forward_from"`
}

// JetBrains is roles/jetbrains' keys.
type JetBrains struct {
	// Backends are the product codes of the backends the role installs.
	Backends []string `yaml:"jetbrains_backends"`
	// Prune removes the other builds of each product installed.
	Prune bool `yaml:"jetbrains_prune"`
}

// Network is roles/network's keys.
type Network struct {
	// DirectLinkInterface is the cable's port. Empty, the role sets up no direct link.
	DirectLinkInterface string `yaml:"network_direct_link_interface"`
	// DirectLinkAddress is the server's address on the cable, with its prefix length.
	DirectLinkAddress string `yaml:"network_direct_link_address"`
}

// Security is roles/security's keys.
type Security struct {
	// Outbound turns on the outbound table: internet and ikeward0 alone.
	Outbound bool `yaml:"security_outbound"`
}
