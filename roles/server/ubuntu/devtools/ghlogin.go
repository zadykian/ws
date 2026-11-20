package devtools

import (
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// TokenFile is the token that gh logs in with, on its last line, which mise writes from TokenVar.
const TokenFile = "/etc/ws/gh-token" //nolint:gosec // G101: a file's path, no credential

// TokenVar is the secret input that mise writes TokenFile from, and the variable that the secrets
// task's run sets to it.
const TokenVar = "WS_GITHUB_TOKEN" //nolint:gosec // G101: a variable's name, no credential

// RunToken returns the token that gh's login takes, empty where there is none. mise writes
// TokenFile in a run before any unit, but in a plan writes nothing. So where lookup gives the
// input, as in the secrets task's run, the token is its own, and a plan sees what the run writes.
// An empty one is no file. Without it, or with a nil lookup, TokenFile's last line gives it.
func RunToken(m effect.Machine, lookup func(string) (string, bool)) (string, error) {
	if lookup != nil {
		if text, ok := lookup(TokenVar); ok {
			return lastLine(text), nil
		}
	}
	text, err := m.ReadFile(TokenFile)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return lastLine(string(text)), err
}

// lastLine returns text's last line, as the file's tail -n 1 gives it, without its spaces.
func lastLine(text string) string {
	text = strings.TrimRight(text, "\n")
	return strings.TrimSpace(text[strings.LastIndex(text, "\n")+1:])
}
