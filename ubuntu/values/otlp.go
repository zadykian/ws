package values

// The collector's way to SigNoz's intake, which SigNoz publishes on the host's loopback.
const (
	// SignozIntake is SigNoz's OTLP/gRPC intake on loopback, on the host and in a self-contained
	// guest. It stays open to any local process there: Docker publishes a port and no socket.
	SignozIntake = "127.0.0.1:14317"
	// OTLPSocket is where a guest's collector reaches the host's intake, in a directory that the
	// image holds: the proxy device makes none (ubuntu/roles/incus/README.md).
	OTLPSocket = "/var/lib/ws/otlp/signoz.sock"
	// OtelcolID is the user ID and the group ID of otelcol-contrib in every guest. The profile names
	// them by number for the socket's owner, so ws makes the account before the .deb does.
	OtelcolID = 800
)

// OTLPEndpoint is the exporter's endpoint of the collector's config: the intake on loopback where
// the guest runs SigNoz, else the socket that the profile's device makes.
func (v Values) OTLPEndpoint() string {
	if v.SelfContained {
		return SignozIntake
	}
	return "unix://" + OTLPSocket
}

// OTLPEndpoint is the host's exporter endpoint: its own SigNoz, on loopback.
func (Host) OTLPEndpoint() string { return SignozIntake }

// Sensors reports whether the collector scrapes node_exporter for temperatures and fans: a guest's
// sysfs shows the host's, so a guest would send them under its own name.
func (Values) Sensors() bool { return false }

// Sensors is true on the host, which has the board's sensors.
func (Host) Sensors() bool { return true }
