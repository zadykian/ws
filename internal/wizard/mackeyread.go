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

// listed reports whether the agent at the socket lists the key that the text of its .pub holds.
func (k *MacKey) listed(ctx context.Context, text, socket string) bool {
	want := firstFields(text)
	if want == "" {
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

// fingerprint returns the fingerprint ssh-keygen prints for the file, or nothing.
func (k *MacKey) fingerprint(ctx context.Context, file string) string {
	out, err := k.M.Run(ctx, "ssh-keygen", "-l", "-f", file)
	if f := strings.Fields(string(out)); err == nil && len(f) >= 2 {
		return f[1]
	}
	return ""
}

// same reports whether the key and its .pub are there and share a fingerprint. ssh-keygen reads the
// key by a link in the lock, with no .pub beside it: by the key's name, it would read the .pub
// instead. On macOS, it can't reread the key from /dev/stdin, whose opens share one offset.
func (k *MacKey) same(ctx context.Context, key string) bool {
	for _, f := range []string{key, key + ".pub"} {
		if info, err := k.M.Stat(f); err != nil || !info.Mode.IsRegular() {
			return false
		}
	}
	link := k.Lock + "/key"
	if _, err := k.M.Run(ctx, "ln", "-sf", key, link); err != nil {
		return false
	}
	fp := k.fingerprint(ctx, link)
	_, _ = k.M.Run(ctx, "rm", "-f", link)
	return fp != "" && fp == k.fingerprint(ctx, key+".pub")
}
