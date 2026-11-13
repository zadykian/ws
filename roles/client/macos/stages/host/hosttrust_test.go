package host

import (
	"context"
	"github.com/zadykian/ws/engine/core/remote"
	"strings"
	"testing"
)

func TestHostKeyTrust(t *testing.T) {
	w := remote.Way{Host: "10.1.1.2", Port: "22"}
	for name, tc := range map[string]struct {
		env  map[string]string
		tty  bool
		want string // in the error, or empty
	}{
		"yes":         {map[string]string{"WS_HOST_KEY": "yes"}, true, ""},
		"fingerprint": {map[string]string{"WS_HOST_KEY": "x " + hostFP}, true, ""},
		"other":       {map[string]string{"WS_HOST_KEY": "SHA256:x"}, true, "'SHA256:x' is neither yes nor " + hostFP},
		"no terminal": {nil, false, "no terminal is there to compare it with ws's console: set WS_HOST_KEY"},
	} {
		h, _ := newHost(tc.env)
		h.Tty = tc.tty
		err := h.trust(context.Background(), w, hostFP)
		if (err == nil) != (tc.want == "") || (err != nil && !strings.Contains(err.Error(), tc.want)) {
			t.Errorf("%s: trust = %v; want %q", name, err, tc.want)
		}
	}
}
