package secrets_test

import (
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/secrets"
)

// roles is where the Mac's tree holds the server's roles.
const roles = secrets.Tree + "/ubuntu/roles"

// withRoles is a machine whose tree holds the roles named, each with its mise.secrets.toml where
// files has one.
func withRoles(names []string, files map[string]string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs[roles] = 0o755
	for _, name := range names {
		m.Dirs[roles+"/"+name] = 0o755
	}
	for name, text := range files {
		m.Files[roles+"/"+name+"/"+secrets.InputsFile] = []byte(text)
	}
	return m
}

func TestInputsTakeEachEnvOnce(t *testing.T) {
	m := withRoles([]string{"devtools", "ikeward", "shell"}, map[string]string{
		"devtools": "[bootstrap.secrets]\n# gh's token.\nWS_GITHUB_TOKEN = { env = \"WS_GITHUB_TOKEN\", allow_empty = true }\n\n" +
			"[bootstrap.files.\"/etc/ws/gh-token\"]\ntemplate = true\ncontent = '{{ secret(name=\"WS_GITHUB_TOKEN\") }}'\n",
		"ikeward": "[bootstrap.secrets]\nWS_IKEWARD_PASSWORD = \"WS_IKEWARD_PASSWORD\"\n" +
			"WS_IKEWARD_P12 = { env = \"WS_IKEWARD_P12\", allow_empty = true }\n",
		"shell": "[bootstrap.secrets]\ngh = { env = \"WS_GITHUB_TOKEN\", allow_empty = true }\n",
	})
	m.Files[roles+"/README.md"] = []byte("# roles\n")
	got, err := secrets.Inputs(m, roles)
	if err != nil {
		t.Fatal(err)
	}
	// One string, as a scan for secrets reads a quoted name after a token's as a key.
	want := strings.Fields("WS_GITHUB_TOKEN WS_IKEWARD_P12 WS_IKEWARD_PASSWORD")
	if !slices.Equal(got, want) {
		t.Errorf("Inputs gives %v, want %v", got, want)
	}
}

func TestInputsRefuseABadName(t *testing.T) {
	m := withRoles([]string{"shell"}, map[string]string{
		"shell": "[bootstrap.secrets]\nendpoint = { env = \"otel-endpoint\" }\n",
	})
	_, err := secrets.Inputs(m, roles)
	if err == nil || !strings.Contains(err.Error(), roles+"/shell/"+secrets.InputsFile) ||
		!strings.Contains(err.Error(), "endpoint") {
		t.Errorf("a name of lowercase gives %v, want an error naming the file and the input", err)
	}
}

func TestInputsRefuseAnInputWithoutEnv(t *testing.T) {
	for _, text := range []string{
		"[bootstrap.secrets]\nTOKEN = { allow_empty = true }\n",
		"[bootstrap.secrets]\nTOKEN = 1\n",
	} {
		m := withRoles([]string{"git"}, map[string]string{"git": text})
		_, err := secrets.Inputs(m, roles)
		if err == nil || !strings.Contains(err.Error(), roles+"/git/"+secrets.InputsFile) ||
			!strings.Contains(err.Error(), "TOKEN") {
			t.Errorf("%q gives %v, want an error naming the file and the input", text, err)
		}
	}
}

func TestInputsSkipARoleWithoutTheFile(t *testing.T) {
	m := withRoles([]string{"docker", "git"}, map[string]string{
		"git": "[bootstrap.secrets]\nWS_GIT_TOKENS = { env = \"WS_GIT_TOKENS\", allow_empty = true }\n",
	})
	got, err := secrets.Inputs(m, roles)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"WS_GIT_TOKENS"}) {
		t.Errorf("Inputs gives %v, want WS_GIT_TOKENS alone", got)
	}

	if _, err := secrets.Inputs(effecttest.NewMachine(), roles); err == nil || !strings.Contains(err.Error(), roles) {
		t.Errorf("a tree without the roles gives %v, want an error naming %s", err, roles)
	}
}

// TestTheTreesInputsParse reads the inputs that this checkout's server roles declare, as the Mac
// reads them from /opt/ws. A tree that declares none passes.
func TestTheTreesInputsParse(t *testing.T) {
	tree := os.DirFS("../../../ubuntu/roles")
	files, err := fs.Glob(tree, "*/"+secrets.InputsFile)
	if err != nil {
		t.Fatal(err)
	}
	m := withRoles(nil, nil)
	for _, file := range files {
		data, err := fs.ReadFile(tree, file)
		if err != nil {
			t.Fatal(err)
		}
		m.Dirs[roles+"/"+path.Dir(file)] = 0o755
		m.Files[roles+"/"+file] = data
	}
	got, err := secrets.Inputs(m, roles)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) > 0 && len(got) == 0 {
		t.Errorf("%d files of inputs declare no input", len(files))
	}
}
