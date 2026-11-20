package secrets_test

import (
	"strings"
	"testing"
)

func TestTaskRefusesAHostnameThatNamesNoProfile(t *testing.T) {
	f := newFixture(t)
	for _, host := range []string{"a/b", "", "Ws", "ws.lan", "nano", "-ws", strings.Repeat("a", 64)} {
		f.env["FAKE_HOST"] = host
		code, out := f.run()
		f.refused(code, out, "one label")
	}
}
