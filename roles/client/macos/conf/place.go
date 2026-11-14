package conf

// defaultPort is the UDP port of wg0 where WS_PORT is empty, and forwardPort the TCP port that a
// router forwards to ws's 22 where WS_REMOTE_PORT is.
const (
	defaultPort = 51820
	forwardPort = 22022
)
