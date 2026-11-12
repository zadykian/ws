package devtools

// Gopls returns where go install puts gopls for the home. Zed and Fresh name it by this path.
func Gopls(home string) string { return home + "/go/bin/gopls" }

// Roslyn returns where dotnet tool install --global puts roslyn-language-server for the home.
func Roslyn(home string) string { return home + "/.dotnet/tools/roslyn-language-server" }

// ServerPaths returns, for the home, the binary of each language server that an editor names by
// path. Its keys are the servers' names.
func ServerPaths(home string) map[string]string {
	return map[string]string{
		"gopls":                       Gopls(home),
		"roslyn-language-server":      Roslyn(home),
		"postgres-language-server":    "/usr/local/bin/postgres-language-server",
		"docker-language-server":      "/usr/local/bin/docker-language-server",
		"ansible-language-server":     serverBin(home, "ansible-language-server"),
		"vscode-html-language-server": serverBin(home, "vscode-html-language-server"),
		"dockerfile-language-server":  serverBin(home, "docker-langserver"),
	}
}

// Dlv returns where go install puts Delve for the home.
func Dlv(home string) string { return home + "/go/bin/dlv" }
