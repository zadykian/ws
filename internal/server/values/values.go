// Package values holds the machine's values, ws's host_vars.yml, as one typed struct. The file
// is parsed once; the playbook reads the same file, so a key that no role knows is refused here
// and there.
package values

// Values is every key of the file. Each role's keys sit in a struct of its name, inlined, so that
// a role's steps read v.Role.Key. A key left out of the file keeps the default that Default gives.
type Values struct {
	Shared    `yaml:",inline"`
	Base      Base      `yaml:",inline"`
	Claude    Claude    `yaml:",inline"`
	Devtools  Devtools  `yaml:",inline"`
	Docker    Docker    `yaml:",inline"`
	Fans      Fans      `yaml:",inline"`
	Helix     Helix     `yaml:",inline"`
	Ikeward   Ikeward   `yaml:",inline"`
	JetBrains JetBrains `yaml:",inline"`
	Network   Network   `yaml:",inline"`
	Security  Security  `yaml:",inline"`
	Shell     Shell     `yaml:",inline"`
	Signoz    Signoz    `yaml:",inline"`
	Wireguard Wireguard `yaml:",inline"`
}

// Shared holds the keys that group_vars/all.yml shares between the roles.
type Shared struct {
	// User is the user the roles set the machine up for.
	User string `yaml:"ws_user"`
	// Home is the user's home. Left empty, it follows User: /root for root, else /home/USER.
	Home string `yaml:"ws_home"`
	// GitSigningKey is the SSH key that signs git's commits, with its public key beside it as
	// .pub. Left empty, it is Home/.ssh/id_ed25519_signing.
	GitSigningKey string `yaml:"ws_git_signing_key"`
}
