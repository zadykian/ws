package wizard

import "github.com/zadykian/ws/internal/conf"

// sequence asks the keys of the Answers stage in order. The machine that ws runs on comes first, as
// the ways to ws follow it.
type sequence struct {
	// vals holds the keys' values so far: mac.conf's, then the answers.
	vals map[string]string
	// lacks reports whether mac.conf has no line for the key.
	lacks func(key string) bool
	// ask gets the value for a key that mac.conf lacks.
	ask func(q question) (string, error)
	// role gives the shell role's default of that name, which ws's git takes.
	role func(name string) string
}

// get returns the key's value: mac.conf's, or the answer where it has no line.
func (s *sequence) get(q question) (string, error) {
	if !s.lacks(q.key) {
		return s.vals[q.key], nil
	}
	v, err := s.ask(q)
	if err == nil {
		s.vals[q.key] = v
	}
	return v, err
}

// run asks the keys in order. git's email is asked only with a name, and needs a value then, and the
// Keychain item only with another repository. git's identity defaults to ws's own.
func (s *sequence) run() error {
	if err := s.ways(); err != nil {
		return err
	}
	name, err := s.get(question{"WS_GIT_NAME", "git's name on the Mac", s.role("shell_git_name"), ""})
	if err != nil {
		return err
	}
	if name != "" {
		if _, err := s.get(question{"WS_GIT_EMAIL", "git's email on the Mac", s.role("shell_git_email"),
			"WS_GIT_NAME and WS_GIT_EMAIL go together"}); err != nil {
			return err
		}
	}
	for _, q := range []question{
		{"WS_TUNNEL_NETWORKS", "More networks through the tunnel, split by blanks", "", ""},
		{"WS_RESOLVERS", "Nameservers behind the tunnel, as DOMAIN=ADDRESS, split by blanks", "", ""},
		{"WS_HOOK", "A release of another repository whose command runs on ws, as OWNER/REPO@vX.Y.Z", "", ""},
	} {
		if _, err := s.get(q); err != nil {
			return err
		}
	}
	if s.vals["WS_HOOK"] == "" {
		return nil
	}
	_, err = s.get(question{"WS_HOOK_SECRET", "The Keychain item of its command's secret", "", ""})
	return err
}

// ways asks the machine, then the ways to it (docs/mac-targets.md). A cloud VM has its name or
// address and its SSH port alone, and a VM on the Mac the Mac's network for its VMs and its address
// there. On the LAN, ws's address is asked only with a LAN, and the router's port only with its
// name. They default to the router's lease where the LAN holds it, and its forward (docs/router.md).
// A cloud VM's name and port, and each address where asked, need a value.
func (s *sequence) ways() error {
	target, err := s.get(question{"WS_TARGET", "The machine ws runs on: lan, cloud or mac-vm", "lan",
		"WS_TARGET needs lan, cloud or mac-vm"})
	if err != nil {
		return err
	}
	var qs []question
	switch target {
	case "cloud":
		need := "WS_TARGET=cloud needs WS_REMOTE and WS_REMOTE_PORT"
		qs = []question{{"WS_REMOTE", "The VM's name or address on the internet", "", need},
			{"WS_REMOTE_PORT", "The VM's SSH port", "22", need}}
	case "mac-vm":
		lan, err := s.get(question{"WS_LAN", "The Mac's network for its VMs", "192.168.64.0/24", "WS_TARGET=mac-vm needs WS_LAN"})
		if err != nil || lan == "" {
			return err
		}
		qs = []question{{"WS_LAN_ADDRESS", "The VM's address on that network", "", "WS_LAN and WS_LAN_ADDRESS go together"}}
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
func (s *sequence) lanWays() error {
	lan, err := s.get(question{"WS_LAN", "ws's home LAN", "10.88.0.0/24", ""})
	if err != nil {
		return err
	}
	if lan != "" {
		def := ""
		if _, errs := conf.Parse("WS_LAN=" + lan + "\nWS_LAN_ADDRESS=10.88.0.10\n"); len(errs) == 0 {
			def = "10.88.0.10"
		}
		if _, err := s.get(question{"WS_LAN_ADDRESS", "ws's address on the LAN", def, "WS_LAN and WS_LAN_ADDRESS go together"}); err != nil {
			return err
		}
	}
	remote, err := s.get(question{"WS_REMOTE", "The router's name on the internet", "", ""})
	if err != nil || remote == "" {
		return err
	}
	_, err = s.get(question{"WS_REMOTE_PORT", "The router's port forwarded to ws's 22", "22022", ""})
	return err
}
