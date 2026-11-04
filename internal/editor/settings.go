package editor

import (
	"embed"
	"fmt"
	"regexp"
)

//go:embed settings.yaml filetypes.yaml
var dataFiles embed.FS

// Settings are the values that every editor takes.
type Settings struct {
	Font  string `yaml:"font"`
	Theme Theme  `yaml:"theme"`
}

// Theme names the dark and the light theme, and which of them runs.
type Theme struct {
	Mode  string `yaml:"mode"`
	Dark  string `yaml:"dark"`
	Light string `yaml:"light"`
}

// FileType gives a language the files that its own patterns miss.
type FileType struct {
	Language string   `yaml:"language"`
	Globs    []string `yaml:"globs"`
}

var languageName = regexp.MustCompile(`^[a-z][a-z-]*$`)

// LoadSettings reads the embedded settings.
func LoadSettings() (Settings, error) {
	var settings Settings
	return settings, load("settings.yaml", &settings)
}

// LoadFileTypes reads the embedded file types, in their order.
func LoadFileTypes() ([]FileType, error) {
	var types []FileType
	if err := load("filetypes.yaml", &types); err != nil {
		return nil, err
	}
	for _, t := range types {
		if !languageName.MatchString(t.Language) || len(t.Globs) == 0 {
			return nil, fmt.Errorf("filetypes.yaml: bad entry %q", t.Language)
		}
	}
	return types, nil
}

func load(name string, out any) error {
	data, err := dataFiles.ReadFile(name)
	if err != nil {
		return err
	}
	if err := decodeStrict(data, out); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
