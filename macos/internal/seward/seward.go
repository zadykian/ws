// Package seward names the files of seward's keys on the Mac, each below the user's home as
// ~/.ssh/NAME. They are the login key, the signing key and the agent's socket.
package seward

// AuthPub is the .pub of the login key that ssh offers ws, and SignPub that of the key that git
// signs with. Socket is the agent that holds both.
const (
	AuthPub = "~/.ssh/ws-auth.pub"
	SignPub = "~/.ssh/ws-sign.pub"
	Socket  = "~/.ssh/seward.sock"
)
