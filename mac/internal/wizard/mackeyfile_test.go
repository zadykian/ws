package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func TestMacKeyFileKey(t *testing.T) {
	for name, tc := range map[string]struct {
		files map[string]string
		fp    string // the .pub's fingerprint, where the key's is aa
		want  string
	}{
		"neither":  {nil, "", "would make /h/k"},
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
		cs, err := k.fileKey(context.Background(), "/h/k")
		got := ""
		if err != nil {
			got = err.Error()
		}
		for _, c := range cs {
			got += c.String()
		}
		if !strings.Contains(got, tc.want) || (tc.want == "" && got != "") {
			t.Errorf("%s: fileKey = %q, want %q", name, got, tc.want)
		}
	}
}
