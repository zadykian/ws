package effect

import (
	"os"
	"path/filepath"
)

// confined opens Confine as a root when path lies under it, and gives path inside it. A nil root
// means path is outside, or nothing is confined. The caller closes the root.
func (m OSMachine) confined(path string) (*os.Root, string, error) {
	if m.Confine == "" {
		return nil, "", nil
	}
	rel, err := filepath.Rel(m.Confine, path)
	if err != nil || !filepath.IsLocal(rel) {
		return nil, "", nil
	}
	root, err := os.OpenRoot(m.Confine)
	return root, rel, err
}
