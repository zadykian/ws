package setup

import "testing"

func TestCldVersion(t *testing.T) {
	tests := []struct {
		out  string
		want string
		ok   bool
	}{
		{"cld 0.13.0\n", "0.13.0", true},
		{"0.6.10", "0.6.10", true},
		{"cld 0.13.0-dirty", "0.13.0-dirty", false},
		{"cld dev", "dev", false},
		{"", "", false},
	}
	for _, tt := range tests {
		if got, ok := cldVersion(tt.out); got != tt.want || ok != tt.ok {
			t.Errorf("cldVersion(%q) = %q, %v; want %q, %v", tt.out, got, ok, tt.want, tt.ok)
		}
	}
}

func TestCldOlder(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"0.13.0", "0.14.0", true},
		{"0.9.0", "0.10.0", true},
		{"1.0.0", "0.99.9", false},
		{"0.13.0", "0.13.0", false},
		{"0.13.1", "0.13.0", false},
	}
	for _, tt := range tests {
		if got := cldOlder(tt.a, tt.b); got != tt.want {
			t.Errorf("cldOlder(%q, %q) = %v; want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
