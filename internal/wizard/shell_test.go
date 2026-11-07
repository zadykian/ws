package wizard

import "testing"

func TestShellWord(t *testing.T) {
	for word, want := range map[string]string{
		"status":                     "status",
		"--only=network,wireguard":   "--only=network,wireguard",
		"https://example.test/a+b@c": "https://example.test/a+b@c",
		"":                           "''",
		"signoz up":                  "'signoz up'",
		"it's":                       `'it'\''s'`,
		"$HOME":                      "'$HOME'",
		"a;b":                        "'a;b'",
	} {
		if got := shellWord(word); got != want {
			t.Errorf("shellWord(%q) = %q, want %q", word, got, want)
		}
	}
}
