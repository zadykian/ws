package stages

import (
	"context"
	"strconv"
	"strings"

	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// tunnelPairs gives the host_vars pairs that bring wg0 to the Mac's want: wireguard_address and
// wireguard_port, each where host_vars differs. A server that goes back to the defaults gets the
// default pair where host_vars holds another value. Where host_vars holds none and the want is the
// default, the run plans nothing, and a first server's run keeps its bytes.
func tunnelPairs(ctx context.Context, ws values.Store, w session.Want) ([]string, error) {
	def := values.Default().Wireguard
	type want struct{ key, value string }
	var wants []want
	if w.Address.IsValid() {
		wants = append(wants, want{"wireguard_address", w.Address.String()})
	}
	if port := w.Port; port != "" || w.Address.IsValid() {
		if port == "" {
			port = strconv.Itoa(def.Port)
		}
		wants = append(wants, want{"wireguard_port", port})
	}
	if len(wants) == 0 {
		return nil, nil
	}
	f, err := values.Form{Ws: ws}.Load(ctx)
	if err != nil {
		return nil, err
	}
	held := map[string]string{}
	for _, field := range f.Fields() {
		held[field.Key] = field.Value
	}
	var pairs []string
	for _, p := range wants {
		if held[p.key] != p.value {
			pairs = append(pairs, p.key+"="+p.value)
		}
	}
	return pairs, nil
}

// naming says what a run writes before it starts, as "wireguard_address: A and wireguard_port: P".
func naming(pairs []string) string {
	named := make([]string, len(pairs))
	for i, p := range pairs {
		named[i] = strings.Replace(p, "=", ": ", 1)
	}
	return strings.Join(named, " and ")
}
