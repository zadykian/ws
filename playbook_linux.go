//go:build linux

package ws

import "embed"

// Playbook is the playbook that the server's ws runs. go:embed drops names that start with . or
// _, and links and modes, so the roles' copies set modes. `all:collections` embeds
// collections/.gitignore, so a tree with no tarball still builds. It holds no host_vars: those
// are a machine's values.
//
//go:embed site.yml ansible.cfg inventory.ini requirements.yml group_vars all:collections
var Playbook embed.FS
