package devtools

import "slices"

// The versions and sources devtools installs from.
const (
	// GoVersion is the Go in /usr/local/go.
	GoVersion = "1.27.3"
	// nodeMajor is Node's major version: NodeSource's repository node_MAJOR.x.
	nodeMajor = "24"
)

// dotnetSDKs are the .NET SDKs. Ubuntu's archive has those of ubuntuSDKs for 26.04, and Launchpad's
// dotnet/backports PPA the others, which the role adds only where dotnetSDKs names one.
var (
	dotnetSDKs = []string{"8.0", "10.0"}
	ubuntuSDKs = []string{"10.0"}
)

// Repository is a third-party apt repository: a deb822 .sources file, with its key in
// /etc/apt/keyrings. The role removes both files of a repository it no longer needs. A key fetched
// again must hold one certificate, whose primary fingerprint is given here: a renewed key keeps it,
// another key, in its place or beside it, does not.
type Repository struct {
	Name, Key, Fingerprint, URIs, Suites, Components string
	Needed                                           bool
}

// repositories returns the repositories for the Ubuntu release, by its codename.
func repositories(release string) []Repository {
	backports := false
	for _, sdk := range dotnetSDKs {
		backports = backports || !slices.Contains(ubuntuSDKs, sdk)
	}
	return []Repository{
		// Ubuntu's archive has an older Node.
		{
			Name: "nodesource", Key: "https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key",
			Fingerprint: "6F71F525282841EEDAF851B42F59B5F99B1BE0B4",
			URIs:        "https://deb.nodesource.com/node_" + nodeMajor + ".x", Suites: "nodistro",
			Components: "main", Needed: true,
		},
		// Ubuntu's archive has an older glow.
		{
			Name: "charm", Key: "https://repo.charm.sh/apt/gpg.key",
			Fingerprint: "ED927B38BE981E53CA09153D03BBF595D4DFD35C",
			URIs:        "https://repo.charm.sh/apt/", Suites: "*", Components: "*", Needed: true,
		},
		// The PPA's key from Ubuntu's keyserver, by the fingerprint Launchpad gives it.
		{
			Name:        "dotnet-backports",
			Key:         "https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x45A3F127159BE9E5017811C62125B164E8E5D3FA",
			Fingerprint: "45A3F127159BE9E5017811C62125B164E8E5D3FA",
			URIs:        "https://ppa.launchpadcontent.net/dotnet/backports/ubuntu/", Suites: release,
			Components: "main", Needed: backports,
		},
	}
}

// replacedFiles are the same repositories as their own instructions add them: NodeSource's setup
// script, Charm's README and add-apt-repository. apt refuses two sources of a repository whose keys
// differ, so the role removes these, keys included.
func replacedFiles(release string) []string {
	return []string{
		"/etc/apt/sources.list.d/nodesource.list",
		"/usr/share/keyrings/nodesource.gpg",
		"/etc/apt/sources.list.d/charm.list",
		"/etc/apt/keyrings/charm.gpg",
		"/etc/apt/sources.list.d/dotnet-ubuntu-backports-" + release + ".sources",
	}
}
