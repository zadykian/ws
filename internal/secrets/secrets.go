// Package secrets holds the values that the roles take from ws's secrets file: age-encrypted TOML,
// a table per kind, decrypted in ws's process. The Mac's commands and ws server apply share this
// format. Nothing here prints a value: an error names a kind, a name, a key or a line.
package secrets

// Secrets is the decrypted file. The names of its fields are those of the tables.
type Secrets struct {
	// Token holds the tokens that ws knows by name.
	Token Tokens
	// Env is the content of ~/.secrets/env: names to values.
	Env map[string]string
	// Git is git's credentials, by the URL of the server, besides GitHub's.
	Git map[string]Git
	// File holds the ikeward profile's files, whole.
	File Files
	// Password holds the passwords that go with the files.
	Password Passwords
	// Whole is true where the run was given a secrets file, as ws server apply sets it: the file
	// is then the truth of what it names, and a role removes what it lacks. The file does not hold
	// it.
	Whole bool
	// Unread is true where a command reads the machine and takes no secrets, as ws server state does:
	// a role then shows what it can, where a run would stop at what the secrets lack.
	Unread bool
}

// Tokens are the tokens of the services that ws logs in to.
type Tokens struct {
	GitHub      string
	AzureDevOps string
}

// Git is one server's credential in git's store.
type Git struct {
	Token         string
	Username      string
	ProactiveAuth bool
}

// Files are the ikeward profile's files, as bytes. The file holds them in base64.
type Files struct {
	VPNP12   []byte
	VPNSswan []byte
}

// Passwords go with the files; the .p12's is empty where it has none.
type Passwords struct {
	VPNP12 string
}

// document is the file as TOML reads and writes it: the files are strings, in base64.
type document struct {
	Token    tokenTable        `toml:"token,omitempty"`
	Env      map[string]string `toml:"env,omitempty"`
	Git      map[string]gitTab `toml:"git,omitempty"`
	File     fileTable         `toml:"file,omitempty"`
	Password passwordTable     `toml:"password,omitempty"`
}

type tokenTable struct {
	GitHub      string `toml:"github,omitempty"`
	AzureDevOps string `toml:"azure_devops,omitempty"`
}

type gitTab struct {
	Token         string `toml:"token,omitempty"`
	Username      string `toml:"username,omitempty"`
	ProactiveAuth bool   `toml:"proactive_auth,omitempty"`
}

type fileTable struct {
	VPNP12   string `toml:"vpn_p12,omitempty"`
	VPNSswan string `toml:"vpn_sswan,omitempty"`
}

type passwordTable struct {
	VPNP12 string `toml:"vpn_p12,omitempty"`
}
