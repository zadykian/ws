package values

import (
	"github.com/zadykian/ws/engine/core/effect"
)

// Check reads values.yml at mac and each server's file beside it. It returns every refusal once,
// in its own file's words: a bad key, a missing file, and what two servers share. A server's
// refusal names the server and the way out of its file, as the facts' do. A Mac with no server yet
// has values.yml alone to check.
func Check(m effect.Machine, mac string) error {
	return ReadFleet(m, mac).Refusal()
}
