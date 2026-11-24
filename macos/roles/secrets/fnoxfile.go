package secrets

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"slices"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/core/effect"
)

const (
	// SharedFile is the secrets repository's file of the values that every server shares.
	SharedFile = "fnox.toml"
	// SharedProvider is SharedFile's age provider. A profile's provider takes another name, as
	// providers merge by name and a profile's would replace this one.
	SharedProvider = "age"
	// NanoLabel is the Nano's recipient's label in each provider's table of recipients. A server's
	// label is its hostname.
	NanoLabel = "nano"
	// BackupLabel is the backup key's label beside it.
	BackupLabel = "backup"
)

// Config is what ws reads of a fnox file: its provider's recipients by label, and its secrets'
// names, sorted. It reads no value.
type Config struct {
	Recipients map[string]string
	Secrets    []string
}

// ReadConfig reads a fnox file of the clone, whose recipients are provider's. It reports false
// where the file is missing. A provider that is not age's, or whose recipients are a list, stops
// it: ws edits a recipient by its label.
func ReadConfig(m effect.Machine, file, provider string) (Config, bool, error) {
	data, err := m.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, false, nil
	}
	if err != nil {
		return Config{}, false, fmt.Errorf("cannot read %s: %w", file, err)
	}
	conf, err := parseConfig(data, provider)
	if err != nil {
		return Config{}, false, fmt.Errorf("%s: %w", file, err)
	}
	return conf, true, nil
}

// parseConfig reads a fnox file's text, whose recipients are provider's.
func parseConfig(data []byte, provider string) (Config, error) {
	var conf struct {
		Providers map[string]struct {
			Type       string `toml:"type"`
			Recipients any    `toml:"recipients"`
		} `toml:"providers"`
		Secrets map[string]any `toml:"secrets"`
	}
	if err := toml.Unmarshal(data, &conf); err != nil {
		return Config{}, err
	}
	p, ok := conf.Providers[provider]
	if !ok {
		return Config{}, fmt.Errorf("no provider %s, [providers.%s]", provider, provider)
	}
	if p.Type != "age" {
		return Config{}, fmt.Errorf("the provider %s is of type %q, not age", provider, p.Type)
	}
	recipients, err := labelled(p.Recipients)
	if err != nil {
		return Config{}, fmt.Errorf("the provider %s: %w", provider, err)
	}
	names := make([]string, 0, len(conf.Secrets))
	for name := range conf.Secrets {
		names = append(names, name)
	}
	slices.Sort(names)
	return Config{Recipients: recipients, Secrets: names}, nil
}

// labelled reads a provider's recipients as a table of labels. A provider with none has an empty
// table.
func labelled(v any) (map[string]string, error) {
	recipients := map[string]string{}
	switch v := v.(type) {
	case nil:
	case map[string]any:
		for label, r := range v {
			s, ok := r.(string)
			if !ok {
				return nil, fmt.Errorf("its recipient %s is no string", label)
			}
			recipients[label] = s
		}
	case []any:
		return nil, errors.New("its recipients are a list, where ws needs a table of labels, as nano = \"age1...\"")
	default:
		return nil, errors.New("its recipients are no table of labels")
	}
	return recipients, nil
}

// ageRecipient is an age recipient, a plain one or a plugin's: age1 and the key in lowercase
// bech32. Start writes each one into a TOML string, so nothing else may.
var ageRecipient = regexp.MustCompile(`^age1[0-9a-z]+$`)

// Start is the text of a new fnox.toml: the Nano's and the backup's recipients, and no secret
// yet. root stops fnox's search for a file above the clone, and if_missing makes a secret that
// opens to no key fail the run.
func Start(nano, backup string) ([]byte, error) {
	for _, r := range []string{nano, backup} {
		if !ageRecipient.MatchString(r) {
			return nil, fmt.Errorf("%q is no age recipient, age1 and lowercase bech32", r)
		}
	}
	return fmt.Appendf(nil, `# The values that every server shares, which ws macos secrets init started. A server's
# own values are in fnox.HOSTNAME.toml, under its provider host.
root = true
if_missing = "error"

[providers.%[1]s]
type = "age"

[providers.%[1]s.recipients]
%[2]s = "%[3]s"
%[4]s = "%[5]s"

[secrets]
`, SharedProvider, NanoLabel, nano, BackupLabel, backup), nil
}
