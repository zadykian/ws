package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
)

// specArgv is a ws command in a spec's argv form: a list that opens with the running ws, by self
// or by its path, then the side.
var specArgv = regexp.MustCompile(`\[\s*"(?:\{\{ self \}\}|/usr/local/bin/ws)",\s*"((?:macos|ubuntu)"(?:,\s*"[^"]*")*)\s*\]`)

// wordArgv is a ws command in a file's text: the side, then the words that name its commands.
var wordArgv = regexp.MustCompile(`\bws ((?:macos|ubuntu)(?: [a-z][a-z-]*)*)`)

// quoted is one string of an argv.
var quoted = regexp.MustCompile(`"([^"]*)"`)

// commandFiles are the files outside the specs that a machine runs ws from.
var commandFiles = []string{
	"../ubuntu/roles/fans/files/ws-fans.service",
	"../ubuntu/roles/git/files/tokens.gitconfig.tmpl",
	"../ubuntu/guest/mise-tasks/secrets",
}

// TestSpecsRunCommandsThatExist finds each ws command that a role's spec or a file runs in the
// tree of commands. No other test resolves a unit's argv, so a command renamed alone breaks a
// machine and the jobs that set one up.
func TestSpecsRunCommandsThatExist(t *testing.T) {
	p := fakeProcess()
	root := cli.Root(p, macosCmd(p), ubuntuCmd(p), unitCmd(p))
	specs, err := filepath.Glob("../*/roles/*/units.toml.tmpl")
	if err != nil || len(specs) == 0 {
		t.Fatalf("specs = %q, err = %v; want the roles' specs", specs, err)
	}
	seen := map[string]int{}
	for _, file := range slices.Concat(specs, commandFiles) {
		text, err := os.ReadFile(file) //nolint:gosec // the spec files of this tree
		if err != nil {
			t.Fatal(err)
		}
		var argvs [][]string
		if strings.HasSuffix(file, "units.toml.tmpl") {
			for _, m := range specArgv.FindAllStringSubmatch(string(text), -1) {
				argvs = append(argvs, quotedWords(m[1]))
			}
		} else {
			for _, m := range wordArgv.FindAllStringSubmatch(string(text), -1) {
				argvs = append(argvs, strings.Fields(m[1]))
			}
		}
		for _, argv := range argvs {
			seen[argv[0]]++
			if err := resolves(root, argv); err != "" {
				t.Errorf("%s: ws %s: %s", file, strings.Join(argv, " "), err)
			}
		}
	}
	for _, side := range []string{"macos", "ubuntu"} {
		if seen[side] == 0 {
			t.Errorf("no ws %s command found in the specs: the pattern reads none", side)
		}
	}
}

// quotedWords returns the strings of an argv's text.
func quotedWords(text string) []string {
	var words []string
	for _, m := range quoted.FindAllStringSubmatch(`"`+text, -1) {
		words = append(words, m[1])
	}
	return words
}

// resolves returns why argv is no command of root: none, a group that words follow, or a command
// of no leaf's. Words after a leaf are its arguments, and root.Find leaves them out.
func resolves(root *cobra.Command, argv []string) string {
	found, rest, err := root.Find(argv)
	switch {
	case err != nil:
		return err.Error()
	case found == root:
		return "no command of the tree"
	case found.HasSubCommands() && len(rest) > 0 && !strings.HasPrefix(rest[0], "-"):
		return "\"" + rest[0] + "\" is no command of ws " + strings.TrimPrefix(found.CommandPath(), "ws ")
	}
	return ""
}
