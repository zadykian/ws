package cli

import (
	"testing"

	"github.com/zadykian/ws/roles/client/macos/tunnel"
)

func TestEmbeddedFilesAreNotEmpty(t *testing.T) {
	for name, data := range map[string][]byte{
		"tunnel.Plist": tunnel.Plist,
	} {
		if len(data) == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}
