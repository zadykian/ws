// Package devtools names the language servers' paths, which Zed's and Fresh's files hold. The role
// itself is mise config and units, which set the servers up at these paths.
package devtools

// The links that units.toml.tmpl makes to mise's tools, root's, so any user runs them. Zed and Fresh
// name them by these paths.
const (
	GoplsPath    = "/usr/local/bin/gopls"
	DlvPath      = "/usr/local/bin/dlv"
	pglsPath     = "/usr/local/bin/postgres-language-server"
	dockerLSPath = "/usr/local/bin/docker-language-server"
)

// Roslyn returns where dotnet tool install --global puts roslyn-language-server for the home.
func Roslyn(home string) string { return home + "/.dotnet/tools/roslyn-language-server" }

// npmBin returns a bin of the npm servers, which units.toml.tmpl installs below the home.
func npmBin(home, name string) string {
	return home + "/.local/lib/language-servers/node_modules/.bin/" + name
}

// ServerPaths returns, for the home, the binary of each language server that an editor names by
// path. Its keys are the servers' names.
func ServerPaths(home string) map[string]string {
	return map[string]string{
		"gopls":                       GoplsPath,
		"roslyn-language-server":      Roslyn(home),
		"postgres-language-server":    pglsPath,
		"docker-language-server":      dockerLSPath,
		"ansible-language-server":     npmBin(home, "ansible-language-server"),
		"vscode-html-language-server": npmBin(home, "vscode-html-language-server"),
		"dockerfile-language-server":  npmBin(home, "docker-langserver"),
	}
}
