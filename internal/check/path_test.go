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
		{"hook", Hook("zadykian/cld@v1.0.0"), true},
		{"hook dots", Hook("a.b/c_d.e@v1.0.0"), true},
		{"hook ws", Hook("zadykian/WS@v1.0.0"), false},
		{"hook ws prefix", Hook("zadykian/ws2@v1.0.0"), true},
		{"hook no slash", Hook("cld@v1.0.0"), false},
		{"hook two slashes", Hook("a/b/c@v1.0.0"), false},
		{"hook slash first", Hook("/a/b@v1.0.0"), false},
		{"hook slash last", Hook("a/@v1.0.0"), false},
		{"hook dot owner", Hook("./b@v1.0.0"), false},
		{"hook dotdot owner", Hook("../b@v1.0.0"), false},
		{"hook dot repo", Hook("a/.@v1.0.0"), false},
		{"hook dotdot repo", Hook("a/..@v1.0.0"), false},
		{"hook colon", Hook("a:b/c@v1.0.0"), false},
		{"hook tag", Hook("a/b@v1.0.0"), true},
		{"hook no tag", Hook("a/b"), false},
		{"hook tag not a release", Hook("a/b@main"), false},
		{"hook empty tag", Hook("a/b@"), false},
		{"hook two tags", Hook("a/b@v1.0.0@v1.0.1"), false},
		{"hook tag on ws", Hook("a/ws@v1.0.0"), false},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}
