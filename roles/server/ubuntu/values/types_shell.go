package values

// Git is the git role's keys.
type Git struct {
	// Clones are the repositories cloned once each.
	Clones []GitClone `yaml:"WS_GIT_CLONES"`
}

// GitClone is a repository, without a token, and where ws clones it.
type GitClone struct {
	Repo string `yaml:"repo,omitempty"`
	Dest string `yaml:"dest,omitempty"`
}

// Signoz is the signoz role's keys.
type Signoz struct {
	URL           string   `yaml:"WS_SIGNOZ_URL"`
	RetentionDays int      `yaml:"WS_SIGNOZ_RETENTION_DAYS"`
	Dashboards    []string `yaml:"WS_SIGNOZ_DASHBOARDS"`
}

// Tunnel is the tunnel role's keys.
type Tunnel struct {
	Address string       `yaml:"WS_TUNNEL_ADDRESS"`
	Port    int          `yaml:"WS_TUNNEL_PORT"`
	Peers   []TunnelPeer `yaml:"WS_TUNNEL_PEERS"`
}

// TunnelPeer is a peer by its name and its address in the tunnel.
type TunnelPeer struct {
	Name    string `yaml:"name,omitempty"`
	Address string `yaml:"address,omitempty"`
}
