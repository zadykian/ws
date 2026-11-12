package wizard

import (
	"context"
	"strings"

	"github.com/zadykian/ws/internal/sshcfg"
)

// listed reports whether the agent at the socket lists the key. The zero key is listed by none.
func (k *MacKey) listed(ctx context.Context, key sshcfg.PublicKey, socket string) bool {
	if key == (sshcfg.PublicKey{}) {
		return false
	}
	out, err := k.M.Run(ctx, "env", "SSH_AUTH_SOCK="+socket, "ssh-add", "-L")
	if err != nil {
		return false
	}
	for _, l := range strings.Split(string(out), "\n") {
		if got, err := sshcfg.ParsePub(l); err == nil && got == key {
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
