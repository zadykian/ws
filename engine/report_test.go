package engine

import (
	"bytes"
	"testing"

	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
)

func TestWriteRoles(t *testing.T) {
	states := []role.RoleState{{Role: "base", Steps: []step.Report{
		{Step: "Packages", Items: []step.Item{{Kind: step.PackageItem, Name: "git", Want: "installed", Found: "missing"}},
			Changes: []step.Pending{{Kind: step.Install, Text: "install git"}}},
		{Step: "Config", Items: []step.Item{{Kind: step.FileItem, Name: "/etc/x", Want: "written", Found: "differs",
			Conflict: "changed by hand", Diff: "-a\n+b\n"}}, Changes: []step.Pending{{Text: "write /etc/x\n-a\n+b\n"}}},
		{Step: "Later", After: "Config"},
	}}}
	var b bytes.Buffer
	if err := WriteRoles(&b, states); err != nil {
		t.Fatal(err)
	}
	want := `base
  Packages
    package git: found missing, want installed
    pending: install git
  Config
    file /etc/x: found differs, want written
      conflict: changed by hand
      -a
      +b
    pending: write /etc/x
  Later
    not read: after Config
`
	if b.String() != want {
		t.Errorf("got\n%s\nwant\n%s", b.String(), want)
	}
}

func TestRoleSectionsNameTheRoles(t *testing.T) {
	reports := []step.Report{{Step: "Packages"}}
	got := RoleSections([]role.RoleState{{Role: "base", Steps: reports}, {Role: "docker"}})
	if len(got) != 2 || got[0].Name != "base" || len(got[0].Reports) != 1 || got[0].Reports[0].Step != "Packages" ||
		got[1].Name != "docker" {
		t.Errorf("sections = %+v", got)
	}
}
