package devtools

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"runtime"
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

// downloadOf returns the one Download that the step for the server plans on goarch.
func downloadOf(t *testing.T, goarch, name string, r release, url func(version, arch string) string) host.Download {
	t.Helper()
	parts, err := download(effecttest.NewMachine(), goarch, name, r, url).Parts(t.Context())
	if err != nil || len(parts) != 1 {
		t.Fatalf("%s %s: %v, %v", name, goarch, parts, err)
	}
	return parts[0].(host.Download)
}

func TestEachServerIsRootsBinaryAtMode0755WithItsPin(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		pg := downloadOf(t, arch, "postgres-language-server", pgls, pglsURL)
		dk := downloadOf(t, arch, "docker-language-server", dockerLS, dockerLSURL)
		if pg.URL != pglsURL(pgls.version, arch) || pg.SHA256 != pgls.sha256[arch] ||
			pg.Path != "/usr/local/bin/postgres-language-server" {
			t.Errorf("%s: %+v", arch, pg)
		}
		if dk.URL != dockerLSURL(dockerLS.version, arch) || dk.SHA256 != dockerLS.sha256[arch] ||
			dk.Path != "/usr/local/bin/docker-language-server" {
			t.Errorf("%s: %+v", arch, dk)
		}
		for _, d := range []host.Download{pg, dk} {
			if d.Mode != 0o755 || d.UID != 0 || d.GID != 0 {
				t.Errorf("%s: owner or mode of %+v", arch, d)
			}
		}
	}
}

func TestAnArchitectureWithNoPinFailsThePlan(t *testing.T) {
	s := download(effecttest.NewMachine(), "riscv64", "docker-language-server", dockerLS, dockerLSURL)
	if _, err := s.Plan(t.Context()); err == nil {
		t.Error("no error for riscv64")
	}
}

// servePins serves a stand-in body at each pinned URL and pins its SHA-256 for the test's length.
func servePins(t *testing.T, m *effecttest.Machine) {
	body := []byte("stand-in binary")
	sum := fmt.Sprintf("%x", sha256.Sum256(body))
	pgls0, dockerLS0 := pgls, dockerLS
	t.Cleanup(func() { pgls, dockerLS = pgls0, dockerLS0 })
	pgls = release{version: pgls0.version, sha256: map[string]string{runtime.GOARCH: sum}}
	dockerLS = release{version: dockerLS0.version, sha256: map[string]string{runtime.GOARCH: sum}}
	m.Pages[pglsURL(pgls.version, runtime.GOARCH)] = body
	m.Pages[dockerLSURL(dockerLS.version, runtime.GOARCH)] = body
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
