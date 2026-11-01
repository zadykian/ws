package uapi

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/zadykian/ws/mac/internal/tunnel"
	"github.com/zadykian/ws/mac/internal/wgkey"
)

// Info is what the device reports of the peer.
type Info struct {
	Stats    tunnel.Stats
	Endpoint netip.AddrPort // invalid where the peer has none
}

// ParseGet reads the text of IpcGet for the peer key, in base64. Info.Stats.Known is false where
// the device has no such peer.
func ParseGet(text, key string) (Info, error) {
	want, err := wgkey.Hex(key)
	if err != nil {
		return Info{}, err
	}
	var info Info
	var sec, nsec int64
	in := false
	for _, line := range strings.Split(text, "\n") {
		name, value, _ := strings.Cut(line, "=")
		if name == "public_key" {
			in = value == want
			info.Stats.Known = info.Stats.Known || in
			continue
		}
		if !in {
			continue
		}
		switch name {
		case "endpoint":
			info.Endpoint, err = netip.ParseAddrPort(value)
		case "last_handshake_time_sec":
			sec, err = strconv.ParseInt(value, 10, 64)
		case "last_handshake_time_nsec":
			nsec, err = strconv.ParseInt(value, 10, 64)
		case "rx_bytes":
			info.Stats.Received, err = strconv.ParseUint(value, 10, 64)
		case "tx_bytes":
			info.Stats.Sent, err = strconv.ParseUint(value, 10, 64)
		}
		if err != nil {
			return Info{}, fmt.Errorf("uapi: %s is '%s'", name, value)
		}
	}
	if sec > 0 || nsec > 0 {
		info.Stats.Handshake = time.Unix(sec, nsec)
	}
	return info, nil
}
