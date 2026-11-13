package host

import (
	"context"
	"strings"

	"github.com/zadykian/ws/engine/core/remote"
)

// wayCall is the ssh -G call that Way makes.
func wayCall(s remote.SSH) string {
	return "ssh " + strings.Join(append([]string{"-G"}, s.Argv(context.Background(), []string{s.Host})...), " ")
}
