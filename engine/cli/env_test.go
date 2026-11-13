package cli

import (
	"os"
	"testing"
)

func TestEnvLockDirFallsBackToTmp(t *testing.T) {
	for _, tc := range []struct{ tmpdir, want string }{{"", "/tmp"}, {"/var/x", "/var/x"}} {
		p := Env{Getenv: func(k string) string {
			if k == "TMPDIR" {
				return tc.tmpdir
			}
			return ""
		}}
		if got := p.LockDir(); got != tc.want {
			t.Errorf("TMPDIR %q: LockDir() = %q, want %q", tc.tmpdir, got, tc.want)
		}
	}
}

func TestReleasesURL(t *testing.T) {
	for set, want := range map[string]string{"": "https://github.com/zadykian/ws/releases", "http://127.0.0.1:22080": "http://127.0.0.1:22080"} {
		if got := releasesURL(func(string) string { return set }); got != want {
			t.Errorf("WS_RELEASES_URL=%q: releasesURL = %q, want %q", set, got, want)
		}
	}
}

func TestIsTerminalIsNotDevNull(t *testing.T) {
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = null.Close() }()
	if isTerminal(null) {
		t.Error("isTerminal(/dev/null) = true")
	}
}
