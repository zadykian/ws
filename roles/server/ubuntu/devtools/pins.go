package devtools

import (
	"slices"

	"github.com/zadykian/ws/roles/server/ubuntu/devtools/internal/aptrepo"
)

// The versions and sources devtools installs from.
const (
	// GoVersion is the Go in /usr/local/go.
	GoVersion = "1.27.3"
	// nodeMajor is Node's major version: NodeSource's repository node_MAJOR.x.
	nodeMajor = "24"
	// goplsVersion is gopls, which go install builds with devtools' Go.
	goplsVersion = "0.23.0"
	// dlvVersion is Delve, which go install builds with devtools' Go. Delve 1.27.2 supports Go 1.25
	// to 1.27 (pkg/goversion/compat.go:14-17 at v1.27.2): a Go 1.28 needs a newer Delve.
	dlvVersion = "1.27.2"
	// roslynVersion is roslyn-language-server, C#'s language server, from nuget.org, where every
	// version is a prerelease. The version alone pins it, as nuget.org refuses a version's second
	// upload. From 5.13.0-1.26514.10 the server defaults telemetry to all (roslyn#86066): a bump adds
	// --telemetryLevel off to every command that starts it.
	roslynVersion = "5.12.0-1.26475.2"
)

// dotnetSDKs are the .NET SDKs. Ubuntu's archive has those of ubuntuSDKs for 26.04, and Launchpad's
// dotnet/backports PPA the others, which the role adds only where dotnetSDKs names one.
var (
	dotnetSDKs = []string{"8.0", "10.0"}
	ubuntuSDKs = []string{"10.0"}
)

// repositories returns the repositories for the Ubuntu release, by its codename.
func repositories(release string) []aptrepo.Repository {
	backports := false
	for _, sdk := range dotnetSDKs {
		backports = backports || !slices.Contains(ubuntuSDKs, sdk)
	}
	return []aptrepo.Repository{
		// Ubuntu's archive has an older Node.
		{
			Name: "nodesource", Key: "https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key",
			Fingerprints: []string{"6F71F525282841EEDAF851B42F59B5F99B1BE0B4"},
			URIs:         "https://deb.nodesource.com/node_" + nodeMajor + ".x", Suites: "nodistro",
			Components: "main", Needed: true,
		},
		// Ubuntu's archive has an older glow.
		{
			Name: "charm", Key: "https://repo.charm.sh/apt/gpg.key",
			Fingerprints: []string{"ED927B38BE981E53CA09153D03BBF595D4DFD35C"},
			URIs:         "https://repo.charm.sh/apt/", Suites: "*", Components: "*", Needed: true,
		},
		// The PPA's key from Ubuntu's keyserver, by the fingerprint Launchpad gives it.
		{
			Name:         "dotnet-backports",
			Key:          "https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x45A3F127159BE9E5017811C62125B164E8E5D3FA",
			Fingerprints: []string{"45A3F127159BE9E5017811C62125B164E8E5D3FA"},
			URIs:         "https://ppa.launchpadcontent.net/dotnet/backports/ubuntu/", Suites: release,
			Components: "main", Needed: backports,
		},
		// Ubuntu's archive has a gh that GitHub's API has left behind. The key file holds a
		// certificate that expired in 2026 beside the one that signs: both are pinned.
		{
			Name: "github-cli", Key: "https://cli.github.com/packages/githubcli-archive-keyring.asc",
			Fingerprints: []string{
				"2C6106201985B60E6C7AC87323F3D4EA75716059", "7F38BBB59D064DBCB3D84D725612B36462313325",
			},
			URIs: "https://cli.github.com/packages", Suites: "stable", Components: "main", Needed: true,
		},
	}
}

// replacedFiles are the same repositories as their own instructions add them: NodeSource's setup
// script, Charm's README, add-apt-repository and GitHub CLI's install guide. The guide's key has
// gone in each of three places. apt refuses two sources of a repository whose keys differ, so the
// role removes these, keys included.
func replacedFiles(release string) []string {
	return []string{
		"/etc/apt/sources.list.d/nodesource.list",
		"/usr/share/keyrings/nodesource.gpg",
		"/etc/apt/sources.list.d/charm.list",
		"/etc/apt/keyrings/charm.gpg",
		"/etc/apt/sources.list.d/dotnet-ubuntu-backports-" + release + ".sources",
		"/etc/apt/sources.list.d/github-cli.list",
		"/etc/apt/keyrings/githubcli-archive-keyring.gpg",
		"/usr/share/keyrings/githubcli-archive-keyring.gpg",
		"/etc/apt/trusted.gpg.d/githubcli-archive-keyring.gpg",
	}
}
