package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrimLogStartsTheLogAnewPastTheLimit(t *testing.T) {
	for _, tt := range []struct {
		name string
		size int
		want int
	}{{"at the limit", LogLimit, LogLimit}, {"past the limit", LogLimit + 1, 0}} {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig("")
			r.Log = filepath.Join(t.TempDir(), "ws-tunnel.log")
			if err := os.WriteFile(r.Log, []byte(strings.Repeat("x", tt.size)), 0o600); err != nil {
				t.Fatal(err)
			}
			r.trimLog()
			fi, err := os.Stat(r.Log)
			if err != nil || fi.Size() != int64(tt.want) {
				t.Errorf("size = %v, err = %v, want %d", fi, err, tt.want)
			}
			if got := strings.Contains(r.log.String(), "started the log anew past 1 MiB"); got != (tt.want == 0) {
				t.Errorf("log = %q", r.log.String())
			}
		})
	}
}

func TestTrimLogLeavesAMissingLogAlone(t *testing.T) {
	r := newRig("")
	r.Log = filepath.Join(t.TempDir(), "none.log")
	r.trimLog()
	r.Log = ""
	r.trimLog()
	if r.log.Len() != 0 {
		t.Errorf("log = %q, want none", r.log.String())
	}
}
