package secrets

import (
	"encoding/base64"

	"go.yaml.in/yaml/v3"
)

// vars are the secrets as the playbook's extra variables, under the names that its roles read.
// The profile's files go as base64, as extra variables hold text.
type vars struct {
	Shell       `yaml:",inline"`
	P12Password string `yaml:"ikeward_p12_password,omitempty"`
	P12         string `yaml:"ikeward_p12_base64,omitempty"`
	Sswan       string `yaml:"ikeward_sswan_base64,omitempty"`
}

// Vars returns the YAML that gives the secrets to the roles that the playbook still runs. It
// holds every value, so it goes by a pipe and never to a file.
func (s Secrets) Vars() ([]byte, error) {
	b64 := func(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
	v := vars{Shell: s.Shell, P12Password: s.Ikeward.P12Password}
	if s.Files.P12 != nil {
		v.P12 = b64(s.Files.P12)
	}
	if s.Files.Sswan != nil {
		v.Sswan = b64(s.Files.Sswan)
	}
	return yaml.Marshal(v)
}
