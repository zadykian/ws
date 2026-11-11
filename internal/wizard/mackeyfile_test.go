package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestMacKeyFileKey(t *testing.T) {
	for name, tc := range map[string]struct {
		files map[string]string
		fp    string // the .pub's fingerprint, where the key's is aa
		want  string
	}{
		"neither":  {nil, "", "/h/k.pub, named in mac.conf's WS_IDENTITY_PUB_KEY, is missing"},
		"both":     {map[string]string{"/h/k": "p", "/h/k.pub": pubText}, "aa", ""},
		"orphan":   {map[string]string{"/h/k.pub": pubText}, "", "is there without its private key"},
		"mismatch": {map[string]string{"/h/k": "p", "/h/k.pub": pubText}, "bb", "is not /h/k's public key"},
	} {
		k, m, _ := newMacKey(t, "")
		for f, v := range tc.files {
			m.Files[f] = []byte(v)
		}
		m.Results["ln -sf /h/k "+lockDir+"/key"] = effecttest.Result{}
		m.Results["rm -f "+lockDir+"/key"] = effecttest.Result{}
		m.Results["ssh-keygen -l -f "+lockDir+"/key"] = effecttest.Result{Out: []byte("256 SHA256:aa me\n")}
		m.Results["ssh-keygen -l -f /h/k.pub"] = effecttest.Result{Out: []byte("256 SHA256:" + tc.fp + " me\n")}
		got := ""
		if err := k.fileKey(context.Background(), "/h/k"); err != nil {
			got = err.Error()
		}
		if !strings.Contains(got, tc.want) || (tc.want == "" && got != "") {
			t.Errorf("%s: fileKey = %q, want %q", name, got, tc.want)
		}
	}
}
