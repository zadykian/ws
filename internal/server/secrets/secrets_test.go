package secrets

import (
	"strings"
	"testing"
)

func TestParseReadsTheKeys(t *testing.T) {
	s, err := Parse([]byte("shell_secret_env: {A: b}\nshell_git_credentials:\n  - {url: 'https://g.example', token: t}\n" +
		"ikeward_p12_file: a.p12.age\nikeward_p12_password: pw\n"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Shell.SecretEnv["A"] != "b" || s.Shell.GitCredentials[0].Token != "t" ||
		s.Ikeward.P12File != "a.p12.age" || s.Ikeward.P12Password != "pw" {
		t.Errorf("got %+v", s)
	}
}

func TestParseRefuses(t *testing.T) {
	for name, in := range map[string]string{
		"unknown key": "shell_secret_envs: {}\n",
		"key twice":   "shell_secret_env: {}\nshell_secret_env: {}\n",
		"files key":   "Files: {}\n",
	} {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestParseEmptyGivesNothing(t *testing.T) {
	s, err := Parse(nil)
	if err != nil || s.Shell.SecretEnv != nil || strings.TrimSpace(s.Ikeward.P12File) != "" {
		t.Errorf("got %+v, %v", s, err)
	}
}
