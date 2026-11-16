package stages

import (
	"context"
	"strconv"
	"strings"

	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// tunnelPairs gives the pairs of values.yml that bring wg0 to the Mac's want: WS_TUNNEL_ADDRESS and
// WS_TUNNEL_PORT, each where values.yml differs. A server that goes back to the defaults gets the
// default pair where values.yml holds another value. Where values.yml holds none and the want is the
// default, the run plans nothing, and a first server's run keeps its bytes.
func tunnelPairs(ctx context.Context, ws values.Store, w session.Want) ([]string, error) {
	def := values.Default().Tunnel
	type want struct{ key, value string }
	var wants []want
	if w.Address.IsValid() {
		wants = append(wants, want{"WS_TUNNEL_ADDRESS", w.Address.String()})
	}
	if port := w.Port; port != "" || w.Address.IsValid() {
		if port == "" {
			port = strconv.Itoa(def.Port)
		}
		wants = append(wants, want{"WS_TUNNEL_PORT", port})
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

// naming says what a run writes before it starts, as "WS_TUNNEL_ADDRESS: A and WS_TUNNEL_PORT: P".
func naming(pairs []string) string {
	named := make([]string, len(pairs))
	for i, p := range pairs {
		named[i] = strings.Replace(p, "=", ": ", 1)
	}
	return strings.Join(named, " and ")
}
