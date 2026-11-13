package resource

import (
	"bytes"
	"fmt"
	"io/fs"
	"text/template"
)

// Render fills the template file name of fsys, usually an embed.FS, with data. A key the data
// lacks is an error, never an empty value.
func Render(fsys fs.FS, name string, data any) ([]byte, error) {
	text, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}
	t, err := template.New(name).Option("missingkey=error").Parse(string(text))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := t.Execute(&out, data); err != nil {
		return nil, fmt.Errorf("render %s: %w", name, err)
	}
	return out.Bytes(), nil
}
