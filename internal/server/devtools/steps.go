package devtools

import (
	_ "embed"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// ghUpgrades lets unattended-upgrades install gh's releases.
//
//go:embed 52unattended-upgrades-github-cli
var ghUpgrades []byte

// Steps returns the role's steps: the apt repositories, the packages from them, gh's telemetry and
// login, Go, the language servers and rustup.
// The packages wait on the repositories, so a check run says "after Devtools repositories" where a
// repository is still to come, as apt may not know the packages then.
func Steps(v values.Values, s secrets.Secrets, m effect.Machine, _ effect.Units, out effect.Out) []step.Step {
	const repositories = "Devtools repositories"
	user := host.User{Name: v.User, Home: v.Home}
	steps := []step.Step{
		host.Group{Title: repositories, Parts: []host.Resource{
			Repositories{M: m},
			host.File{M: m, Path: "/etc/apt/apt.conf.d/52unattended-upgrades-github-cli", Content: ghUpgrades, Mode: 0o644},
		}},
		host.Group{Title: "Devtools packages", After: repositories, Parts: []host.Resource{
			// Node at its major version, from NodeSource's repository; gh from GitHub CLI's;
			// glow from Charm's; each .NET SDK from Ubuntu's archive or the PPA.
			host.Packages{M: m, Out: out, Recommends: true, Names: packageNames()},
			// The libraries go in without recommends: the browser needs the libraries alone.
			host.Packages{M: m, Out: out, Names: v.Devtools.ChromiumLibraries},
		}},
		// gh's telemetry goes off before gh's login runs gh again.
		host.Group{Title: "Devtools gh telemetry", After: "Devtools packages", Parts: []host.Resource{
			GHTelemetry{M: m, User: user},
		}},
		// gh logs in by token.github, after its package.
		host.Group{Title: "Devtools gh login", After: "Devtools packages", Parts: []host.Resource{
			GHLogin{M: m, User: v.User, Home: v.Home, Token: s.Token.GitHub},
		}},
		host.Group{Title: "Devtools Go", Parts: []host.Resource{goTree(m)}},
		host.Group{Title: "Devtools gopls", After: "Devtools Go", Parts: []host.Resource{
			goplsTool{M: m, User: user, Path: Gopls(v.Home)},
		}},
		host.Group{Title: "Devtools Delve", After: "Devtools Go", Parts: []host.Resource{
			delveTool{M: m, User: user, Path: Dlv(v.Home), Version: dlvVersion, GoVersion: GoVersion},
		}},
		host.Group{Title: "Devtools roslyn-language-server", After: "Devtools packages", Parts: []host.Resource{
			roslynTool{M: m, User: user, Path: Roslyn(v.Home)},
		}},
	}
	return append(append(steps, serverSteps(m, user)...), host.Group{Title: "Devtools rustup", Parts: []host.Resource{
		Rustup{M: m, User: v.User, Home: v.Home},
	}})
}

// serverSteps returns the steps of the language servers that editors name by path. They are the
// downloads of the SQL and Docker servers, and the npm servers, which wait on the Node package. They install
// apart from Steps so that a test can run them with no stand-ins for Go and the packages.
func serverSteps(m effect.Machine, user host.User) []step.Step {
	return []step.Step{
		download(m, "postgres-language-server", pgls, pglsURL),
		download(m, "docker-language-server", dockerLS, dockerLSURL),
		user.WithAccount(m, "Devtools npm servers", "Devtools packages", func(a host.Account) ([]host.Resource, error) {
			return serverFiles(m, a)
		}),
	}
}

// packageNames returns the packages of Node, gh, glow and the .NET SDKs.
func packageNames() []string {
	names := []string{"nodejs=" + nodeMajor + ".*", "gh", "glow"}
	for _, sdk := range dotnetSDKs {
		names = append(names, "dotnet-sdk-"+sdk)
	}
	return names
}
