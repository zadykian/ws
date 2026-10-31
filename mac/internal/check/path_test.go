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
		{"socket", Socket("~/.agent", false), true},
		{"socket blank quoted", Socket("~/Library/Group Containers/a.sock", true), true},
		{"socket blank bare", Socket("~/Library/Group Containers/a.sock", false), false},
		{"socket root", Socket("/", false), false},
		{"socket relative", Socket("agent.sock", false), false},
		{"socket glob", Socket("~/a*", false), false},
		{"hook", Hook("zadykian/cld"), true},
		{"hook dots", Hook("a.b/c_d.e"), true},
		{"hook ws", Hook("zadykian/WS"), false},
		{"hook ws prefix", Hook("zadykian/ws2"), true},
		{"hook no slash", Hook("cld"), false},
		{"hook two slashes", Hook("a/b/c"), false},
		{"hook slash first", Hook("/a/b"), false},
		{"hook slash last", Hook("a/"), false},
		{"hook dot owner", Hook("./b"), false},
		{"hook dotdot owner", Hook("../b"), false},
		{"hook dot repo", Hook("a/."), false},
		{"hook dotdot repo", Hook("a/.."), false},
		{"hook colon", Hook("a:b/c"), false},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}
