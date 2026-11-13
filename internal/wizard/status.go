package wizard

import "strings"

// wsStatus is what the remote script's status prints of ws, each line in a field of its own. A
// field is empty where ws has none, which the script prints as -.
type wsStatus struct {
	// Run is the last full run's stamp, zero where none ran.
	Run stamp
	// Active is the unit of the run that runs now.
	Active string
	// Version is what ws's binary prints as its version.
	Version string
	// User is the ws_user that host_vars names.
	User string
	// MacKey is the Mac's key that wg0's peer takes, and WgKey is wg0's own.
	MacKey, WgKey string
	// Signing is the type and base64 of ws's public signing key, or none where WS_USER's git config
	// names no key file.
	Signing string
	// Hostname is ws's.
	Hostname string
	// Claude is ok, none or login: WS_USER's claude has a login, is not there, or has none.
	Claude string
	// FansRefused is set where Secure Boot refuses nct6687, the fans' driver.
	FansRefused bool
	// Cable is the port of the Mac's cable that host_vars and the network role's netplan file name.
	Cable string
	// Peered is set where wg0 has MacKey as a peer.
	Peered bool
	// OutboundOff is set where host_vars turns the security role's outbound table off.
	OutboundOff bool
	// Secrets is the SHA-256 of ws's secrets file, as a run would stamp it, and Vars that of
	// host_vars.
	Secrets, Vars string
}

// stamp is a full run's stamp, which the remote script's job writes as five words: VERSION STATUS
// on|off SECRETSUM VARSSUM. A field is empty where the stamp names -.
type stamp struct {
	// Version is what the run's ws printed as its version before the run.
	Version string
	// Exit is ws server ubuntu apply's status.
	Exit string
	// Off is set where host_vars turned the outbound table off as the run began.
	Off bool
	// Secrets is the SHA-256 of the secrets file that the run read, and Vars that of host_vars as
	// the run began.
	Secrets, Vars string
}

// parseStatus reads the status's lines, NAME VALUE. The first line of a name counts, and a name
// the wizard does not read is left.
func parseStatus(text string) wsStatus {
	seen := map[string]bool{}
	var s wsStatus
	for _, l := range strings.Split(text, "\n") {
		name, value, ok := strings.Cut(l, " ")
		if !ok || seen[name] {
			continue
		}
		seen[name] = true
		s.set(name, none(value))
	}
	return s
}

// set puts the value of the line NAME in its field.
func (s *wsStatus) set(name, value string) {
	text := map[string]*string{"active": &s.Active, "ws": &s.Version, "user": &s.User, "mac.pub": &s.MacKey,
		"wg0": &s.WgKey, "signing": &s.Signing, "hostname": &s.Hostname, "claude": &s.Claude,
		"cable": &s.Cable, "secrets": &s.Secrets, "vars": &s.Vars}
	if field, ok := text[name]; ok {
		*field = value
	}
	switch name {
	case "run":
		s.Run = parseStamp(value)
	case "fans":
		s.FansRefused = value == "refused"
	case "peer":
		s.Peered = value == "ok"
	case "outbound":
		s.OutboundOff = value == "off"
	}
}

// parseStamp reads a stamp of five words, and returns a zero one for any other text.
func parseStamp(text string) stamp {
	f := strings.Fields(text)
	if len(f) != 5 {
		return stamp{}
	}
	return stamp{Version: none(f[0]), Exit: none(f[1]), Off: f[2] == "off", Secrets: none(f[3]),
		Vars: none(f[4])}
}

// none returns the value, or an empty string for -, which the script prints for a missing one.
func none(value string) string {
	if value == "-" {
		return ""
	}
	return value
}
