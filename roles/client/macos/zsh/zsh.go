// Package zsh writes the hook that turns off the terminal modes a dropped ssh leaves on, and the line
// of ~/.zshrc that sources it.
package zsh

import (
	"context"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/roles/internal/etcws"
)

const (
	// hookFile is the hook, root's, which a run replaces whole.
	hookFile = etcws.Dir + "/zshrc"
	// zshLine is the line of ~/.zshrc that sources the hook.
	zshLine = "source " + hookFile
)

// Zsh writes /etc/ws/zshrc, a hook that turns off at each prompt the modes that a dropped ssh
// leaves on (roles/client/macos/ghostty/README.md). It adds the line that sources the hook at the end of ~/.zshrc,
// and keeps the old file as ~/.zshrc.TIME~, as the ssh step keeps ~/.ssh/config. It warns, and leaves
// ~/.zshrc, where ~/.zshrc is a link that lacks the line.
type Zsh struct {
	Home string
	// Hook is roles/client/macos/zsh/files/zshrc.
	Hook []byte
	M    effect.Machine
	Out  effect.Out
	// Now stamps the copy of ~/.zshrc. Nil means time.Now.
	Now func() time.Time
}

func (z *Zsh) rc() string { return z.Home + "/.zshrc" }

// zshSources says whether a line of text is zshLine, spaces aside.
func zshSources(text []byte) bool {
	for _, line := range strings.Split(string(text), "\n") {
		if strings.TrimSpace(line) == zshLine {
			return true
		}
	}
	return false
}

// rcLines is zshLine at the end of ~/.zshrc.
func (z *Zsh) rcLines() file.UserLines {
	return file.UserLines{
		M: z.M, Out: z.Out, Path: z.rc(), Lines: zshLine + "\n", Shown: zshLine, Mode: 0o644, Now: z.Now,
		Has: func(_ context.Context, _ effect.Machine, _ string, text []byte) (bool, error) {
			return zshSources(text), nil
		},
	}
}

// hookParts is /etc/ws and the hook in it, each saying its change as it applies.
func (z *Zsh) hookParts() []resource.Resource {
	var parts []resource.Resource
	for _, r := range etcws.Files(z.M, hookFile, z.Hook) {
		parts = append(parts, resource.Said{Out: z.Out, Part: r})
	}
	return parts
}
