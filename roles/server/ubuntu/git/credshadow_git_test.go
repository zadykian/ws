package git

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// gitMachine runs git for real, in an environment of the test's own, where the fixtures answer by
// tables. Its git reads no config of the caller's.
type gitMachine struct {
	*effecttest.Machine
	home string
}

func (m gitMachine) RunInput(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // the shadow check names git and its arguments
	cmd.Env = []string{"HOME=" + m.home, "PATH=" + os.Getenv("PATH"), "GIT_CONFIG_NOSYSTEM=1"}
	cmd.Stdin = bytes.NewReader(stdin)
	return cmd.Output()
}

// The account's file includes ws's, which includes tokens.gitconfig, whose reset of each token's
// URL comes before ws's program. That section is ws's, so no note names it. The same
// section in the account's own file is noted.
func TestGitconfigNotesNoSectionOfWsOwnInclude(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	tokens := filepath.Join(dir, "tokens.gitconfig")
	ws := filepath.Join(dir, "gitconfig")
	section := "[credential \"https://git.example.com\"]\n\thelper =\n"
	files := map[string]string{
		tokens: section + "\thelper = !ws\n",
		ws:     "[include]\n\tpath = " + tokens + "\n",
	}
	for path, text := range files {
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	g := shadow{M: gitMachine{effecttest.NewMachine(), dir}, Creds: []Credential{{URL: "https://git.example.com"}}}
	rc := "[include]\n\tpath = " + ws + "\n"
	if notes, err := g.credentials(t.Context(), []byte(rc)); err != nil || len(notes) > 0 {
		t.Errorf("notes of ws's own include: %q, %v; want none", notes, err)
	}
	if notes, err := g.credentials(t.Context(), []byte(rc+section)); err != nil || len(notes) != 1 {
		t.Errorf("notes of the account's own section: %q, %v; want one", notes, err)
	}
}
