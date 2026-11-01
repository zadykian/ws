// Package mac holds the scripts that the wizard sends to ws over ssh, which stay sh: they run
// on ws, where the ws binary is not.
package mac

import _ "embed" // for the scripts' go:embed

// RemoteScript is wizard-remote.sh, which ws mac wizard sends to root on ws, with sh -s.
//
//go:embed wizard-remote.sh
var RemoteScript []byte

// RemoteFilesScript is wizard-remote-files.sh, which the stages of the Mac's keys and hooks send
// to root on ws.
//
//go:embed wizard-remote-files.sh
var RemoteFilesScript []byte
