// Package git reads git's files in /etc/ws, which git's units write, and says whether ~/.gitconfig
// includes ws's settings at its top.
package git

import (
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/internal/etcws"
	"github.com/zadykian/ws/internal/gitcfg"
	"github.com/zadykian/ws/internal/sshcfg"
	"github.com/zadykian/ws/macos/internal/seward"
)

// config returns /etc/ws/gitconfig: its header, and the settings that ws shares with the server.
// ~/.gitconfig includes it at its top, so that its own settings win.
func config(in gitcfg.Inputs) string {
	return file.Header("#") + "\n" +
		"# ~/.gitconfig includes it at its top, so that its own settings win where both set one.\n" +
		gitcfg.Settings(gitcfg.Mac(), in)
}

// wrapper returns the script that gives ssh-keygen seward's agent, which holds the signing key.
// So a terminal and an IDE sign alike, whatever agent the app that runs git has.
func wrapper(home string) string {
	return "#!/bin/sh\n" +
		file.Header("#") + "\n" +
		"# git's gpg.ssh.program: ssh-keygen with seward's agent, which holds the signing key.\n" +
		"SSH_AUTH_SOCK='" + sshcfg.Expand(seward.Socket, home) + "'\n" +
		"export SSH_AUTH_SOCK\n" +
		"exec ssh-keygen \"$@\"\n"
}

// keygen is the script that gives ssh-keygen seward's agent, which git signs through.
const keygen = etcws.Dir + "/ssh-keygen"

// Gh is the gh that git's credential command runs: the copy that git's unit keeps of mise's, in
// /usr/local/bin, which every shell's PATH holds.
const Gh = "/usr/local/bin/gh"

// Texts is git's files in /etc/ws that facts build: gitconfig, and the script that signs through
// seward's agent. The allowed signers hold the key that a unit makes, so a template reads it.
type Texts struct{ Config, Keygen string }

// TextsOf returns git's files in /etc/ws, which sign with seward's key.
func TextsOf(home string) Texts {
	in := gitcfg.Inputs{Key: sshcfg.Expand(seward.SignPub, home), Gh: Gh, Program: keygen}
	return Texts{Config: config(in), Keygen: wrapper(home)}
}
