package wizard

import "github.com/zadykian/ws/mac/internal/conf"

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
