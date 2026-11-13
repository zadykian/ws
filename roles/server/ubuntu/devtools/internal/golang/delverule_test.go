package golang

import (
	"context"
	"testing"
)

func TestDelveCurrent(t *testing.T) {
	const mod = "\tmod\tgithub.com/go-delve/delve\tv1.27.2\th1:abc=\n"
	cases := []struct {
		name string
		out  string
		err  error
		want bool
	}{
		{"current", "/root/go/bin/dlv: go1.27.3\n\tpath\tgithub.com/go-delve/delve/cmd/dlv\n" + mod, nil, true},
		{"no binary", "", context.Canceled, false},
		{"older Go", "/root/go/bin/dlv: go1.26.0\n" + mod, nil, false},
		{"older Delve", "/root/go/bin/dlv: go1.27.3\n\tmod\tgithub.com/go-delve/delve\tv1.27.1\th1:abc=\n", nil, false},
		{"no mod line", "/root/go/bin/dlv: go1.27.3\n\tpath\tgithub.com/go-delve/delve/cmd/dlv\n", nil, false},
	}
	for _, c := range cases {
		if got := delveCurrent([]byte(c.out), c.err, "1.27.3", "1.27.2"); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
