// Package git reads git's files in /etc/ws, which git's units write, and says whether ~/.gitconfig
// includes ws's settings at its top.
package git

import (
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/roles/client/internal/keys"
	"github.com/zadykian/ws/roles/internal/etcws"
	"github.com/zadykian/ws/roles/internal/gitcfg"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// config returns /etc/ws/gitconfig: its header, and the settings that ws shares with the server.
// ~/.gitconfig includes it at its top, so that its own settings win.
func config(in gitcfg.Inputs) string {
	return file.Header("#") + "\n" +
		"# ~/.gitconfig includes it at its top, so that its own settings win where both set one.\n" +
		gitcfg.Settings(gitcfg.Mac(), in)
}

// wrapper returns the script that gives ssh-keygen Secretive's agent, for a key that Secretive
// holds. So a terminal and an IDE sign alike, whatever agent the app that runs git has.
func wrapper(secretive string) string {
	return "#!/bin/sh\n" +
		file.Header("#") + "\n" +
		"# git's gpg.ssh.program: ssh-keygen with Secretive's agent, which holds the signing key.\n" +
		"SSH_AUTH_SOCK='" + secretive + "/socket.ssh'\n" +
		"export SSH_AUTH_SOCK\n" +
		"exec ssh-keygen \"$@\"\n"
}

// keygen is the script that gives ssh-keygen Secretive's agent, which git signs through.
const keygen = etcws.Dir + "/secretive-ssh-keygen"

// Gh is the gh that git's credential command runs: the copy that git's unit keeps of mise's, in
// /usr/local/bin, which every shell's PATH holds.
const Gh = "/usr/local/bin/gh"

// Texts is git's files in /etc/ws: gitconfig, allowed_signers, and the script that signs through
// Secretive's agent, empty for a key in a file.
type Texts struct{ Config, Signers, Keygen string }

// TextsOf returns git's files in /etc/ws for the signing key.
func TextsOf(home string, key keys.NamedKey) Texts {
	in := gitcfg.Inputs{Key: key.File, Gh: Gh}
	t := Texts{Signers: gitcfg.AllowedSigners(gitcfg.Mac(), file.Header("#"), key.Key.String())}
	if sshcfg.SecretiveKey(key.File, home) {
		in.Program = keygen
		t.Keygen = wrapper(home + "/" + sshcfg.SecretiveData)
	}
	t.Config = config(in)
	return t
}
