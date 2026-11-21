package check

import "testing"

func TestResolvers(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", true},
		{"example.internal=10.20.0.53", true},
		{"a.test=10.0.0.1,10.0.0.2,10.0.0.3 b.test=10.0.0.4", true},
		{"a.test=10.0.0.1,10.0.0.2,10.0.0.3,10.0.0.4", false},
		{"a.test=10.0.0.1 a.test=10.0.0.2", false},
		{"A.test=10.0.0.1", false},
		{"a.test", false},
		{"a.test=", false},
		{"a.test=10.0.0.1,", false},
		{"a.test=10.0.0.1,,10.0.0.2", false},
		{"a.test=ns.test", false},
		{"=10.0.0.1", false},
		{"a.test=10.0.0.1=b", false},
		{"a_b.test=10.0.0.1", false},
	}
	for _, tt := range tests {
		if got := Resolvers(tt.in); got != tt.want {
			t.Errorf("Resolvers(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
