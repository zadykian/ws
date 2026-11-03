package zed

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// valuesOf returns the defaults for the user and the home.
func valuesOf(user, home string) values.Values {
	v := values.Default()
	v.User, v.Home = user, home
	return v
}

// servePins serves a stand-in body at each pinned URL and pins its SHA-256 for the test's length.
func servePins(t *testing.T, m *effecttest.Machine) {
	t.Helper()
	body := []byte("stand-in binary")
	sum := fmt.Sprintf("%x", sha256.Sum256(body))
	pgls0, dockerLS0 := pgls, dockerLS
	t.Cleanup(func() { pgls, dockerLS = pgls0, dockerLS0 })
	pgls = release{version: pgls0.version, sha256: map[string]string{runtime.GOARCH: sum}}
	dockerLS = release{version: dockerLS0.version, sha256: map[string]string{runtime.GOARCH: sum}}
	m.Pages[pglsURL(pgls, runtime.GOARCH)] = body
	m.Pages[dockerLSURL(dockerLS, runtime.GOARCH)] = body
}

// run applies the role on a machine with an empty home for the user.
func run(t *testing.T, user, home string) *effecttest.Machine {
	t.Helper()
	m := effecttest.NewMachine()
	m.Dirs[home] = 0o755
	servePins(t, m)
	m.Results["id -u "+user] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g "+user] = effecttest.Result{Out: []byte("1000\n")}
	steps := Steps(valuesOf(user, home), secrets.Secrets{}, m, &effecttest.Out{})
	if _, err := step.Run(context.Background(), steps, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestSettingsAreValidJSONWithTheUsersPaths(t *testing.T) {
	v := values.Default()
	v.Home = "/home/ws"
	got, err := settingsFile(v)
	if err != nil {
		t.Fatal(err)
	}
	_, body, _ := strings.Cut(string(got), "\n")
	if !json.Valid([]byte(body)) {
		t.Fatalf("not JSON:\n%s", body)
	}
	for _, want := range []string{"/home/ws/go/bin/gopls", "/home/ws/.dotnet/tools/roslyn-language-server",
		"/home/ws/go/bin/dlv"} {
		if !strings.Contains(body, `"`+want+`"`) {
			t.Errorf("settings lack %q", want)
		}
	}
}

func TestSettingsAreOwnedByTheUserOnAnEmptyHome(t *testing.T) {
	m := run(t, "ws", "/home/ws")
	for _, p := range []string{"/home/ws/.config/zed/settings.json"} {
		if m.Owners[p] != [2]int{1000, 1000} {
			t.Errorf("%s owner %v", p, m.Owners[p])
		}
	}
	if m.Dirs["/home/ws/.config"] != 0o755 || m.Dirs["/home/ws/.config/zed"] != 0o755 {
		t.Errorf("dirs %v", m.Dirs)
	}
}

func TestASecondRunChangesNothing(t *testing.T) {
	m := run(t, "root", "/root")
	steps := Steps(valuesOf("root", "/root"), secrets.Secrets{}, m, &effecttest.Out{})
	changed, err := step.Run(context.Background(), steps, &effecttest.Out{}, true)
	if err != nil || changed {
		t.Errorf("second run: %v, %v", changed, err)
	}
}

func TestTheLimitNamesTheUserAndTheValue(t *testing.T) {
	m := run(t, "ws", "/home/ws")
	_, line, _ := strings.Cut(string(m.Files[limitsPath]), "\n")
	if line != "ws soft nofile 65536\n" {
		t.Errorf("limits line %q", line)
	}
	if m.Owners[limitsPath] != [2]int{} {
		t.Errorf("owner %v", m.Owners[limitsPath])
	}
}

func TestTheServersAreRootsBinariesAtMode0755(t *testing.T) {
	m := run(t, "ws", "/home/ws")
	for _, p := range []string{"/usr/local/bin/postgres-language-server", "/usr/local/bin/docker-language-server"} {
		if string(m.Files[p]) != "stand-in binary" || m.Owners[p] != [2]int{} {
			t.Errorf("%s: owner %v", p, m.Owners[p])
		}
	}
}
