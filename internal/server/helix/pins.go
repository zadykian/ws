package helix

// The versions that ws installs.
const (
	// goplsVersion is gopls, which go install builds with devtools' Go.
	goplsVersion = "0.23.0"
	// roslynVersion is roslyn-language-server, C#'s language server, from nuget.org, where every
	// version is a prerelease. The version alone pins it, as nuget.org refuses a version's second
	// upload.
	roslynVersion = "5.12.0-1.26475.2"
)
