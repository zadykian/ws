//go:build linux

package ws

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlaybookHoldsTheFiles(t *testing.T) {
	for _, name := range []string{
		"site.yml", "ansible.cfg", "inventory.ini", "requirements.yml",
		"group_vars/all.yml", "roles/security/tasks/main.yml", "collections/.gitignore",
	} {
		if _, err := fs.Stat(Playbook, name); err != nil {
			t.Errorf("Playbook lacks %s: %v", name, err)
		}
	}
	if _, err := fs.Stat(Playbook, "host_vars"); err == nil {
		t.Error("Playbook holds host_vars, a machine's values")
	}
}

func TestPlaybookEqualsTheTree(t *testing.T) {
	for _, dir := range []string{"roles", "group_vars"} {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "__pycache__" {
					return filepath.SkipDir
				}
				return nil
			}
			for _, elem := range strings.Split(path, "/") {
				if strings.HasPrefix(elem, ".") || strings.HasPrefix(elem, "_") {
					t.Errorf("%s has a path element go:embed drops", path)
					return nil
				}
			}
			want, err := fs.ReadFile(os.DirFS("."), path)
			if err != nil {
				return err
			}
			got, err := fs.ReadFile(Playbook, path)
			if err != nil {
				t.Errorf("Playbook lacks %s: %v", path, err)
			} else if !bytes.Equal(got, want) {
				t.Errorf("Playbook's %s differs from the file", path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
