package devtools

// roslynVersion is roslyn-language-server, C#'s language server, from nuget.org, where every
// version is a prerelease. The version alone pins it, as nuget.org refuses a version's second
// upload. From 5.13.0-1.26514.10 the server defaults telemetry to all (roslyn#86066): a bump adds
// --telemetryLevel off to every command that starts it.
const roslynVersion = "5.12.0-1.26475.2"
