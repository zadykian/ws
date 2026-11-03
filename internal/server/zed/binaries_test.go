package zed

import (
	"regexp"
	"testing"
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
