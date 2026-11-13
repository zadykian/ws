package stages

import (
	"context"
	"strconv"

	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// tunnelPairs gives the host_vars pairs that bring wg0 to the Mac's want: wireguard_address where
// the want is not the default's, wireguard_port likewise, each where host_vars differs. A want of
// the defaults reads nothing, so a server at slot 0 on port 51820 plans no pair.
func tunnelPairs(ctx context.Context, ws values.Store, w session.Want) ([]string, error) {
	def := values.Default().Wireguard
	var want [][2]string
	if w.Address.IsValid() && w.Address.String() != def.Address {
		want = append(want, [2]string{"wireguard_address", w.Address.String()})
	}
	if w.Port != "" && w.Port != strconv.Itoa(def.Port) {
		want = append(want, [2]string{"wireguard_port", w.Port})
	}
	if len(want) == 0 {
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
	for _, p := range want {
		if held[p[0]] != p[1] {
			pairs = append(pairs, p[0]+"="+p[1])
		}
	}
	return pairs, nil
}
