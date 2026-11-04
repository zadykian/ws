package zed

import (
	"regexp"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestTheURLsNameEachServersAssetPerArchitecture(t *testing.T) {
	const pg = "https://github.com/supabase-community/postgres-language-server/releases/download/0.28.2/"
	const dk = "https://github.com/docker/docker-language-server/releases/download/v0.20.1/"
	for got, want := range map[string]string{
		pglsURL(pgls, "amd64"):         pg + "postgres-language-server_x86_64-unknown-linux-gnu",
		pglsURL(pgls, "arm64"):         pg + "postgres-language-server_aarch64-unknown-linux-gnu",
		dockerLSURL(dockerLS, "amd64"): dk + "docker-language-server-linux-amd64-v0.20.1",
		dockerLSURL(dockerLS, "arm64"): dk + "docker-language-server-linux-arm64-v0.20.1",
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
func downloadOf(t *testing.T, goarch, name string, r release, url func(release, string) string) host.Download {
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
		if pg.URL != pglsURL(pgls, arch) || pg.SHA256 != pgls.sha256[arch] ||
			pg.Path != "/usr/local/bin/postgres-language-server" {
			t.Errorf("%s: %+v", arch, pg)
		}
		if dk.URL != dockerLSURL(dockerLS, arch) || dk.SHA256 != dockerLS.sha256[arch] ||
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
