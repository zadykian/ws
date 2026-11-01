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
	var doc yaml.Node
	if err := doc.Encode(v); err != nil {
		return nil, err
	}
	unsafe(&doc, false)
	return yaml.Marshal(&doc)
}

// unsafe tags every string that is a value, so that Ansible takes it as written and never as a
// template: a secret may hold {{, {% or {#. A key stays plain.
func unsafe(n *yaml.Node, key bool) {
	switch n.Kind {
	case yaml.ScalarNode:
		if !key && n.Tag == "!!str" {
			n.Tag = "!unsafe"
			n.Style |= yaml.TaggedStyle
		}
	case yaml.MappingNode:
		for i, c := range n.Content {
			unsafe(c, i%2 == 0)
		}
	default:
		for _, c := range n.Content {
			unsafe(c, false)
		}
	}
}
