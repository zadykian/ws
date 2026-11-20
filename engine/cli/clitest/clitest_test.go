package clitest

import "testing"

// TestWithEnvLooksUpTheMap holds a variable set empty apart from one unset, as ikeward's check
// tells a run with the secrets from one without them.
func TestWithEnvLooksUpTheMap(t *testing.T) {
	if v, ok := Env().LookupEnv("A"); v != "" || ok {
		t.Errorf("Env's lookup of A = %q, %v; want unset", v, ok)
	}
	e := WithEnv(Env(), map[string]string{"A": "", "B": "b"})
	for _, c := range []struct {
		name, value string
		set         bool
	}{{"A", "", true}, {"B", "b", true}, {"C", "", false}} {
		if v, ok := e.LookupEnv(c.name); v != c.value || ok != c.set {
			t.Errorf("lookup of %s = %q, %v; want %q, %v", c.name, v, ok, c.value, c.set)
		}
	}
	if got := e.Getenv("B"); got != "b" {
		t.Errorf("Getenv(B) = %q, want b", got)
	}
}
