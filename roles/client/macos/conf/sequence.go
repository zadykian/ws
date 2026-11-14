package conf

import "strconv"

// Sequence asks the keys of the Answers stage in order. The machine that ws runs on comes first, as
// the ways to ws follow it.
type Sequence struct {
	// Vals holds the keys' values so far: mac.conf's, then the answers.
	Vals map[string]string
	// Lacks reports whether mac.conf has no line for the key.
	Lacks func(key string) bool
	// Ask gets the value for a key that mac.conf lacks.
	Ask func(q Question) (string, error)
	// Role gives the shell role's default of that name, which ws's git takes.
	Role func(name string) string
}

// get returns the key's value: mac.conf's, or the answer where it has no line.
func (s *Sequence) get(q Question) (string, error) {
	if !s.Lacks(q.Key) {
		return s.Vals[q.Key], nil
	}
	v, err := s.Ask(q)
	if err == nil {
		s.Vals[q.Key] = v
	}
	return v, err
}

// Run asks the keys in order. git's email is asked only with a name, and needs a value then. git's
// identity defaults to ws's own.
func (s *Sequence) Run() error {
	if err := s.ways(); err != nil {
		return err
	}
	name, err := s.get(Question{"WS_GIT_NAME", "git's name on the Mac", s.Role("shell_git_name"), ""})
	if err != nil {
		return err
	}
	if name != "" {
		if _, err := s.get(Question{"WS_GIT_EMAIL", "git's email on the Mac", s.Role("shell_git_email"),
			"WS_GIT_NAME and WS_GIT_EMAIL go together"}); err != nil {
			return err
		}
	}
	for _, q := range []Question{
		{"WS_TUNNEL_NETWORKS", "More networks through the tunnel, split by blanks", "", ""},
		{"WS_RESOLVERS", "Nameservers behind the tunnel, as DOMAIN=ADDRESS, split by blanks", "", ""},
	} {
		if _, err := s.get(q); err != nil {
			return err
		}
	}
	return nil
}

// ways asks the machine, then the ways to it (roles/client/macos/conf/README.md). A cloud VM has its name or
// address and its SSH port alone, and a VM on the Mac the Mac's network for its VMs and its address
// there. On the LAN, ws's address is asked only with a LAN, and the router's port only with its
// name. They default to the router's lease where the LAN holds it, and its forward
// (roles/server/ubuntu/network/router.md). A cloud VM's name and port, and each address where
// asked, need a value.
func (s *Sequence) ways() error {
	target, err := s.get(Question{"WS_TARGET", "The machine ws runs on", "lan",
		"WS_TARGET needs lan, cloud or mac-vm"})
	if err != nil {
		return err
	}
	var qs []Question
	switch target {
	case "cloud":
		need := "WS_TARGET=cloud needs WS_REMOTE and WS_REMOTE_PORT"
		qs = []Question{{"WS_REMOTE", "The VM's name or address on the internet", "", need},
			{"WS_REMOTE_PORT", "The VM's SSH port", "22", need}}
	case "mac-vm":
		lan, err := s.get(Question{"WS_LAN", "The Mac's network for its VMs", "192.168.64.0/24", "WS_TARGET=mac-vm needs WS_LAN"})
		if err != nil || lan == "" {
			return err
		}
		qs = []Question{{"WS_LAN_ADDRESS", "The VM's address on that network", "", "WS_LAN and WS_LAN_ADDRESS go together"}}
	default:
		return s.lanWays()
	}
	for _, q := range qs {
		if _, err := s.get(q); err != nil {
			return err
		}
	}
	return nil
}

// lanWays asks the ways to a ws on the LAN.
func (s *Sequence) lanWays() error {
	lan, err := s.get(Question{"WS_LAN", "ws's home LAN", "10.88.0.0/24", ""})
	if err != nil {
		return err
	}
	if lan != "" {
		def := ""
		if _, errs := Parse("WS_LAN=" + lan + "\nWS_LAN_ADDRESS=10.88.0.10\n"); len(errs) == 0 {
			def = "10.88.0.10"
		}
		if _, err := s.get(Question{"WS_LAN_ADDRESS", "ws's address on the LAN", def, "WS_LAN and WS_LAN_ADDRESS go together"}); err != nil {
			return err
		}
	}
	remote, err := s.get(Question{"WS_REMOTE", "The router's name on the internet", "", ""})
	if err != nil || remote == "" {
		return err
	}
	_, err = s.get(Question{"WS_REMOTE_PORT", "The router's port forwarded to ws's 22", strconv.Itoa(forwardPort), ""})
	return err
}
