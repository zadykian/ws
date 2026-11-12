package devtools

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"regexp"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/step"
)

func TestTheURLsNameEachServersAssetPerArchitecture(t *testing.T) {
	const pg = "https://github.com/supabase-community/postgres-language-server/releases/download/0.28.2/"
	const dk = "https://github.com/docker/docker-language-server/releases/download/v0.20.1/"
	for got, want := range map[string]string{
		pglsURL(pgls.version, "amd64"):         pg + "postgres-language-server_x86_64-unknown-linux-gnu",
		pglsURL(pgls.version, "arm64"):         pg + "postgres-language-server_aarch64-unknown-linux-gnu",
		dockerLSURL(dockerLS.version, "amd64"): dk + "docker-language-server-linux-amd64-v0.20.1",
		dockerLSURL(dockerLS.version, "arm64"): dk + "docker-language-server-linux-arm64-v0.20.1",
	} {
		if got != want {
			t.Errorf("url %s, want %s", got, want)
		}
	}
}

func TestEachPinHoldsAHexDigestForBothArchitectures(t *testing.T) {
	hex := regexp.MustCompile(`^[0-9a-f]{64}$`)
	for name, r := range map[string]release{"pgls": pgls, "dockerLS": dockerLS} {
		for _, arch := range []string{"amd64", "arm64"} {
			sum, err := r.digest(arch)
			if err != nil || !hex.MatchString(sum) {
				t.Errorf("%s %s: %q, %v", name, arch, sum, err)
			}
		}
	}
}

func TestADigestFailsForAnArchitectureWithNoPin(t *testing.T) {
	if _, err := pgls.digest("riscv64"); err == nil {
		t.Error("no error for riscv64")
	}
}

// pinnedOf returns the one pinned file that the step for the server makes.
func pinnedOf(t *testing.T, name string, r release, url func(version, arch string) string) host.PinnedFile {
	parts := download(effecttest.NewMachine(), name, r, url).Parts
	if len(parts) != 1 {
		t.Fatalf("%s: %v", name, parts)
	}
	return parts[0].(host.PinnedFile)
}

func TestEachServerIsRootsBinaryAtMode0755WithItsPin(t *testing.T) {
	for name, r := range map[string]release{"postgres-language-server": pgls, "docker-language-server": dockerLS} {
		p := pinnedOf(t, name, r, pglsURL)
		if p.Version != r.version || p.Path != "/usr/local/bin/"+name || p.Mode != 0o755 ||
			!maps.Equal(p.SHA256, r.sha256) {
			t.Errorf("%s: %+v", name, p)
		}
	}
}

// servePins serves a stand-in body at each pinned URL for amd64 and pins its SHA-256 for the test's
// length.
func servePins(t *testing.T, m *effecttest.Machine) {
	body := []byte("stand-in binary")
	sum := fmt.Sprintf("%x", sha256.Sum256(body))
	pgls0, dockerLS0 := pgls, dockerLS
	t.Cleanup(func() { pgls, dockerLS = pgls0, dockerLS0 })
	pgls = release{version: pgls0.version, sha256: map[string]string{"amd64": sum}}
	dockerLS = release{version: dockerLS0.version, sha256: map[string]string{"amd64": sum}}
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Pages[pglsURL(pgls.version, "amd64")] = body
	m.Pages[dockerLSURL(dockerLS.version, "amd64")] = body
}

// withPackages returns serverSteps after an empty step of the name that the npm servers wait on.
func withPackages(m *effecttest.Machine, user, home string) []step.Step {
	return append([]step.Step{host.Group{Title: "Devtools packages"}},
		serverSteps(m, host.User{Name: user, Home: home})...)
}

// runServers applies serverSteps on a machine with an empty home for the user, and returns it.
func runServers(t *testing.T, user, home string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs[home] = 0o755
	servePins(t, m)
	m.Results["id -u "+user] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g "+user] = effecttest.Result{Out: []byte("1000\n")}
	m.Files[npmBin] = []byte("stand-in npm")
	m.Results[npmCommand(user, home)] = effecttest.Result{}
	steps := withPackages(m, user, home)
	if _, err := step.Run(t.Context(), steps, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestTheServersAreRootsBinariesAtMode0755(t *testing.T) {
	m := runServers(t, "ws", "/home/ws")
	for _, p := range []string{"/usr/local/bin/postgres-language-server", "/usr/local/bin/docker-language-server"} {
		if string(m.Files[p]) != "stand-in binary" || m.Owners[p] != [2]int{} {
			t.Errorf("%s: owner %v", p, m.Owners[p])
		}
	}
}

func TestTheServersChangeNothingOnASecondRun(t *testing.T) {
	m := runServers(t, "root", "/root")
	changed, err := step.Run(t.Context(), withPackages(m, "root", "/root"), &effecttest.Out{}, true)
	if err != nil || changed {
		t.Errorf("second run: %v, %v", changed, err)
	}
}
