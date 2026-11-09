package wizard

import "strings"

// stamp is a full run's stamp, which the remote script's job writes as five words: VERSION STATUS
// on|off SECRETSUM VARSSUM. A field is empty where the stamp names -.
type stamp struct {
	// Version is what the run's ws printed as its version before the run.
	Version string
	// Exit is ws server apply's status.
	Exit string
	// Off is set where host_vars turned the outbound table off as the run began.
	Off bool
	// Secrets is the SHA-256 of the secrets file that the run read, and Vars that of host_vars as
	// the run began.
	Secrets, Vars string
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
