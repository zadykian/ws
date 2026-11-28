package secrets_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/secrets"
)

// The Mac names the server lab, whose host key hostKey its known_hosts holds where known is set.
// fnox.toml holds the recipients' lines more, and the server answers ssh with its hostname, server.
func TestHostname(t *testing.T) {
	label := func(l, key string) string { return l + ` = "` + key + `"` + "\n" }
	for name, tt := range map[string]struct {
		more, server, want string
		known, asks        bool
	}{
		"a name that differs":    {more: label("ws-a", hostKey), server: "ws-b", want: "ws-a", known: true},
		"no label of the key":    {more: label("ws-a", otherHost), server: "ws-c", want: "ws-c", known: true, asks: true},
		"two labels of the key":  {more: label("ws-a", hostKey) + label("ws-b", hostKey), server: "ws-c", want: "ws-c", known: true, asks: true},
		"no entry for lab":       {more: label("ws-a", hostKey), server: "ws-c", want: "ws-c", asks: true},
		"a label of no profile":  {more: label("local", hostKey), server: "ws-a", known: true},
		"a server of no profile": {server: "Lab Server", known: true, asks: true},
		"a server that fails":    {known: true, asks: true},
		"no fnox.toml":           {more: "-", server: "ws-a", known: true},
	} {
		m := cloneable()
		m.Files[knownHosts] = []byte("lab " + hostKey + "\n")
		m.Results[findLine] = effecttest.Result{Out: []byte("lab " + hostKey + "\n"), Err: exitOne()}
		if tt.known {
			m.Results[findLine] = effecttest.Result{Out: []byte("lab " + hostKey + "\n")}
		}
		if tt.more != "-" {
			m.Files[cloneDir+"/"+secrets.SharedFile] = []byte(sprintf(startText, tt.more))
		}
		asked := false
		ask := func(context.Context) (string, error) {
			asked = true
			if tt.server == "" {
				return "", errors.New("ssh: connect to host lab: refused")
			}
			return tt.server, nil
		}
		got, err := open(t, m).Hostname(context.Background(), "/h", "lab", ask)
		if got != tt.want || (err == nil) != (tt.want != "") || asked != tt.asks {
			t.Errorf("%s: Hostname = %q, %v, asked %t; want %q, asked %t", name, got, err, asked, tt.want, tt.asks)
		}
	}
}
