package signoz

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// Steps returns the role's steps, in the order they run: the files, the password and the up, then
// Sync's. The role takes no secret: the root user's password is drawn on the machine.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, _ effect.Units, out effect.Out) []step.Step {
	return append([]step.Step{Files(m, Dir), Password(m), Up{M: m, Out: out, Dir: Dir}}, Sync(v.Signoz, m, out)...)
}
