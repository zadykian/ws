package wizard

import (
	"context"
	"strings"
)

// firstFields returns the first two fields of the first line of text that holds any: a public key's
// type and its base64.
func firstFields(text string) string {
	for _, l := range strings.Split(text, "\n") {
		if f := strings.Fields(l); len(f) >= 2 {
			return f[0] + " " + f[1]
		}
	}
	return ""
}

// listed reports whether the agent at the socket lists the key of the .pub file.
func (k *MacKey) listed(ctx context.Context, pub, socket string) bool {
	text, err := k.M.ReadFile(pub)
	want := firstFields(string(text))
	if err != nil || want == "" {
		return false
	}
	out, err := k.M.Run(ctx, "env", "SSH_AUTH_SOCK="+socket, "ssh-add", "-L")
	if err != nil {
		return false
	}
	for _, l := range strings.Split(string(out), "\n") {
		if f := strings.Fields(l); len(f) >= 2 && f[0]+" "+f[1] == want {
			return true
		}
	}
	return false
}
