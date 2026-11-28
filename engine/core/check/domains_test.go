package check

import "testing"

func TestDomains(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", true},
		{"corp.test", true},
		{"a.test b.test\tc-d.test", true},
		{"a.test a.test", false},
		{"A.test", false},
		{"a.test,b.test", false},
		{"a.test=10.0.0.1", false},
		{"a_b.test", false},
		{"-a.test", false},
		{"a..test", false},
	}
	for _, tt := range tests {
		if got := Domains(tt.in); got != tt.want {
			t.Errorf("Domains(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
