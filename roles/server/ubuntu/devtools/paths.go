// Package devtools sets up the account's tools that the role's mise config leaves: gh's telemetry
// and login, roslyn-language-server, the npm servers and rustup. It names the language servers'
// paths, which Zed's and Fresh's files hold.
package devtools

import "github.com/zadykian/ws/roles/server/ubuntu/devtools/internal/npm"

// The links that units.toml makes to mise's tools, root's, so any WS_USER runs them. Zed and Fresh
// name them by these paths.
const (
	GoplsPath    = "/usr/local/bin/gopls"
	DlvPath      = "/usr/local/bin/dlv"
	pglsPath     = "/usr/local/bin/postgres-language-server"
	dockerLSPath = "/usr/local/bin/docker-language-server"
)

// Roslyn returns where dotnet tool install --global puts roslyn-language-server for the home.
func Roslyn(home string) string { return home + "/.dotnet/tools/roslyn-language-server" }

// ServerPaths returns, for the home, the binary of each language server that an editor names by
// path. Its keys are the servers' names.
func ServerPaths(home string) map[string]string {
	return map[string]string{
		"gopls":                       GoplsPath,
		"roslyn-language-server":      Roslyn(home),
		"postgres-language-server":    pglsPath,
		"docker-language-server":      dockerLSPath,
		"ansible-language-server":     npm.Bin(home, "ansible-language-server"),
		"vscode-html-language-server": npm.Bin(home, "vscode-html-language-server"),
		"dockerfile-language-server":  npm.Bin(home, "docker-langserver"),
	}
}
