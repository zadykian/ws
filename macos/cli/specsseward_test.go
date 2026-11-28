package cli

import (
	"os"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// TestSewardServiceRunsTheCopy keeps seward's agent a user service that runs the copy in
// /usr/local/bin after mise's tools. Its environment holds the version, so that a new pin changes
// the plist, which restarts the agent.
func TestSewardServiceRunsTheCopy(t *testing.T) {
	text, err := os.ReadFile(macRolesDir + "/ssh/" + macRoleFile)
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Bootstrap struct {
			Services map[string]struct {
				Scope         string            `toml:"scope"`
				Command       string            `toml:"command"`
				RequiresTools bool              `toml:"requires_tools"`
				Environment   map[string]string `toml:"environment"`
			} `toml:"services"`
		} `toml:"bootstrap"`
	}
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatal(err)
	}
	s, ok := c.Bootstrap.Services["seward"]
	if !ok {
		t.Fatal("ssh's mise file has no [bootstrap.services.seward]")
	}
	if s.Scope != "user" || s.Command != "/usr/local/bin/seward agent" || !s.RequiresTools {
		t.Errorf("seward's service = %+v, want a user service of /usr/local/bin/seward agent with requires_tools", s)
	}
	if s.Environment["WS_SEWARD_VERSION"] == "" {
		t.Errorf("seward's service environment = %v, want WS_SEWARD_VERSION, which a new pin changes", s.Environment)
	}
}
