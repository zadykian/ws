package session

import _ "embed" // for the scripts' go:embed

// RemoteScript is wizard-remote.sh, which ws client macos wizard sends to root on ws, with sh -s.
//
//go:embed wizard-remote.sh
var RemoteScript []byte

// RemoteFilesScript is wizard-remote-files.sh, which ws's run, Tunnel and Cable stages send to root
// on ws.
//
//go:embed wizard-remote-files.sh
var RemoteFilesScript []byte
