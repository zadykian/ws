package macos

import "github.com/zadykian/ws/roles/client/macos/internal/change"

// Write is change.Write, which cmd names to print a diff and cannot import from an internal package.
type Write = change.Write

// Backup is change.Backup, which cmd names to leave a backup unsaid.
type Backup = change.Backup
