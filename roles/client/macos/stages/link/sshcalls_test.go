package link

import (
	"context"
	"github.com/zadykian/ws/engine/core/remote"
	"strings"
)

// wayCall is the ssh -G call that Way makes.
func wayCall(s remote.SSH) string {
	return "ssh " + strings.Join(append([]string{"-G"}, s.Argv(context.Background(), []string{s.Host})...), " ")
}

// tunnelCall is the call that Tunnel makes.
func tunnelCall(s remote.SSH, args ...string) string {
	return "ssh " + strings.Join(append(s.Shared("%n"), args...), " ")
}
