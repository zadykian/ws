package check

import "testing"

func TestPath(t *testing.T) {
	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{"file", KeyFile("~/.ssh/ws.pub"), true},
		{"file absolute", KeyFile("/Users/a/k.pub"), true},
		{"file not pub", KeyFile("~/.ssh/ws"), false},
		{"file relative", KeyFile("ssh/ws.pub"), false},
		{"file blank", KeyFile("~/my key.pub"), false},
		{"file tilde twice", KeyFile("~~/k.pub"), false},
		{"socket", Socket("~/.agent"), true},
		{"socket blank", Socket("~/Library/Group Containers/a.sock"), true},
		{"socket root", Socket("/"), false},
		{"socket relative", Socket("agent.sock"), false},
		{"socket glob", Socket("~/a*"), false},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}
