package wizard

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zadykian/ws/engine/core/role"
)

// ErrOtherWs says that ws holds another version of ws, or none, as before the stage that installs
// this one. Another version's report names its own steps, and an old one has no report to give.
var ErrOtherWs = errors.New("ws's report refused")

// State returns the roles of what ws server ubuntu state --json reports on ws, from a ws of this
// version alone. It runs no state op where the status names another, and the report's own version
// decides after that. The op changes nothing, so what the status read stands.
func (s *Session) State(ctx context.Context, version string) ([]role.RoleState, error) {
	st, ok := s.Status(ctx)
	if !ok {
		return nil, errors.New("ws's report: ws's status read nothing")
	}
	if st.Version != version {
		return nil, otherWs(cmp.Or(st.Version, "-"), version)
	}
	out, err := s.do(ctx, opState)
	if err != nil {
		return nil, fmt.Errorf("ws's report: %w", err)
	}
	var doc role.StateDoc
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, fmt.Errorf("ws's report: %w", err)
	}
	if doc.Version != version {
		return nil, otherWs(doc.Version, version)
	}
	return doc.Roles, nil
}

// otherWs is ErrOtherWs, with the version that ws holds, - for none, and this ws's.
func otherWs(held, version string) error {
	return fmt.Errorf("%w: ws holds %s, and this ws is %s", ErrOtherWs, held, version)
}
