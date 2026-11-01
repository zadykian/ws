package values

// Shell is roles/shell's keys.
type Shell struct {
	GitName  string `yaml:"shell_git_name"`
	GitEmail string `yaml:"shell_git_email"`
	// GitSigning signs each commit with the signing key.
	GitSigning bool `yaml:"shell_git_signing"`
	// GitAllowedSigners is the allowed signers file. Empty, it is under Home.
	GitAllowedSigners string `yaml:"shell_git_allowed_signers"`
	// GitClones are the repositories cloned once each.
	GitClones []GitClone `yaml:"shell_git_clones"`
}

// GitCredential is a token for one server, in the secrets file: the values file has no token. State
// is present, the default, or absent.
type GitCredential struct {
	URL           string `yaml:"url,omitempty"`
	Token         string `yaml:"token,omitempty"`
	Username      string `yaml:"username,omitempty"`
	ProactiveAuth bool   `yaml:"proactive_auth,omitempty"`
	State         string `yaml:"state,omitempty"`
}

// GitClone is a repository, without a token, and where it is cloned.
type GitClone struct {
	Repo string `yaml:"repo,omitempty"`
	Dest string `yaml:"dest,omitempty"`
}

// Signoz is roles/signoz's keys.
type Signoz struct {
	Dir           string   `yaml:"signoz_dir"`
	URL           string   `yaml:"signoz_url"`
	RetentionDays int      `yaml:"signoz_retention_days"`
	Dashboards    []string `yaml:"signoz_dashboards"`
}

// Wireguard is roles/wireguard's keys.
type Wireguard struct {
	Address string          `yaml:"wireguard_address"`
	Port    int             `yaml:"wireguard_port"`
	Peers   []WireguardPeer `yaml:"wireguard_peers"`
}

// WireguardPeer is a peer by its name and its address in the tunnel.
type WireguardPeer struct {
	Name    string `yaml:"name,omitempty"`
	Address string `yaml:"address,omitempty"`
}
