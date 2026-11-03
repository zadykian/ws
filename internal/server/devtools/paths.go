package devtools

// Gopls returns where go install puts gopls for the home. helix and Zed name it by this path.
func Gopls(home string) string { return home + "/go/bin/gopls" }

// Roslyn returns where dotnet tool install --global puts roslyn-language-server for the home.
func Roslyn(home string) string { return home + "/.dotnet/tools/roslyn-language-server" }

// Dlv returns where go install puts Delve for the home.
func Dlv(home string) string { return home + "/go/bin/dlv" }
