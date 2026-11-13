package conf

import "testing"

func TestDefault(t *testing.T) {
	defaults := []byte("shell_git_name: Ada\nshell_git_email: ada@example.com\n")
	if got := Default(defaults, "shell_git_email"); got != "ada@example.com" || Default(defaults, "shell_other") != "" {
		t.Errorf("Default = %q", got)
	}
}
