package wizard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zadykian/ws/internal/server/apply"
)

// ErrOtherWs says that a report comes from another version of ws, whose steps need not be this one's.
var ErrOtherWs = errors.New("ws's report refused")

// State returns the roles of what ws server state --json reports on ws, from a ws of this version
// alone. The op changes nothing, so what Status read stands.
func (r *Remote) State(ctx context.Context, version string) ([]apply.RoleState, error) {
	out, err := r.query(ctx, "state")
	if err != nil {
		return nil, fmt.Errorf("ws's report: %w", err)
	}
	var doc apply.StateDoc
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, fmt.Errorf("ws's report: %w", err)
	}
	if doc.Version != version {
		return nil, otherWs(doc.Version, version)
	}
	return doc.Roles, nil
}

// otherWs is ErrOtherWs, with the version that ws holds and this ws's.
func otherWs(held, version string) error {
	return fmt.Errorf("%w: ws holds %s, and this ws is %s", ErrOtherWs, held, version)
}
