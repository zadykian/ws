package devtools

import "embed"

// serversLock holds the npm servers' package.json and the lockfile that pins them. The lock's
// sha512 integrity is the check on each download, as npm ci fails where one differs. To bump a
// version, see docs/devtools.md.
//
//go:embed servers/package.json servers/package-lock.json
var serversLock embed.FS
