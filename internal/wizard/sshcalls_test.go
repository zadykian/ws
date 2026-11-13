package wizard

import (
	"context"
	"strings"
)

// wayCall is the ssh -G call that Way makes.
func wayCall(s SSH) string {
	return "ssh " + strings.Join(append([]string{"-G"}, s.Argv(context.Background(), []string{"ws"})...), " ")
}

// tunnelCall is the call that Tunnel makes.
func tunnelCall(s SSH, args ...string) string {
	return "ssh " + strings.Join(append(s.Shared("%n"), args...), " ")
}
