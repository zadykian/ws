package devtools

// The versions devtools installs from.
const (
	// GoVersion is the Go in /usr/local/go.
	GoVersion = "1.27.3"
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
