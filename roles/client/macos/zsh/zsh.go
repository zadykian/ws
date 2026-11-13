// Package zsh writes the hook that turns off the terminal modes a dropped ssh leaves on, and the line
// of ~/.zshrc that sources it.
package zsh

import (
	"context"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/file"
)

const (
	// zshHeader is the hook's first line, by which a run knows the file for its own.
	zshHeader = "# Written by ws client macos apply: its next run replaces changes made here."
	// zshLine is the line of ~/.zshrc that sources the hook.
	zshLine = "source ~/.config/ws/zshrc"
)

// Zsh writes ~/.config/ws/zshrc, a hook that turns off at each prompt the modes that a dropped ssh
// leaves on (roles/client/macos/ghostty/README.md). It adds the line that sources the hook at the end of ~/.zshrc,
// and keeps the old file as ~/.zshrc.TIME~, as the ssh step keeps ~/.ssh/config. It warns, and leaves
// ~/.zshrc, where the hook is not ws's or ~/.zshrc is a link that lacks the line.
type Zsh struct {
	Home string
	// Hook is roles/client/macos/zsh/files/zshrc, its first line zshHeader.
	Hook []byte
	M    effect.Machine
	Out  effect.Out
	// Now stamps the copy of ~/.zshrc. Nil means time.Now.
	Now func() time.Time
}

func (z *Zsh) hook() string { return z.Home + "/.config/ws/zshrc" }

func (z *Zsh) rc() string { return z.Home + "/.zshrc" }

// zshOwn says whether the hook's first line is ws's.
func zshOwn(have []byte) bool {
	first, _, _ := strings.Cut(string(have), "\n")
	return first == zshHeader
}

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

// hookFile is the hook, mode 644.
func (z *Zsh) hookFile() file.UserFile {
	return file.UserFile{M: z.M, Out: z.Out, Path: z.hook(), Text: z.Hook, Mode: 0o644, Own: zshOwn}
}
