package devtools

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Steps returns the role's steps: the apt repositories, the packages from them, gh's login, Go and rustup.
// The packages wait on the repositories, so a check run says "after Devtools repositories" where a
// repository is still to come, as apt may not know the packages then.
func Steps(v values.Values, s secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	const repositories = "Devtools repositories"
	user := host.User{Name: v.User, Home: v.Home}
	return []step.Step{
		host.Group{Title: repositories, Parts: []host.Resource{Repositories{M: m}}},
		host.Group{Title: "Devtools packages", After: repositories, Parts: []host.Resource{
			// Node at its major version, from NodeSource's repository; gh from Ubuntu's archive;
			// glow from Charm's; each .NET SDK from Ubuntu's archive or the PPA.
			host.Packages{M: m, Out: out, Recommends: true, Names: packageNames()},
			// The libraries go in without recommends: the browser needs the libraries alone.
			host.Packages{M: m, Out: out, Names: v.Devtools.ChromiumLibraries},
		}},
		// gh logs in by token.github, after its package.
		host.Group{Title: "Devtools gh login", After: "Devtools packages", Parts: []host.Resource{
			GHLogin{M: m, User: v.User, Home: v.Home, Token: s.Token.GitHub},
		}},
		host.Group{Title: "Devtools Go", Parts: []host.Resource{Go{M: m}}},
		host.Group{Title: "Devtools gopls", After: "Devtools Go", Parts: []host.Resource{
			goplsTool{M: m, User: user, Path: Gopls(v.Home)},
		}},
		host.Group{Title: "Devtools roslyn-language-server", After: "Devtools packages", Parts: []host.Resource{
			roslynTool{M: m, User: user, Path: Roslyn(v.Home)},
		}},
		host.Group{Title: "Devtools rustup", Parts: []host.Resource{
			Rustup{M: m, User: v.User, Home: v.Home},
		}},
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
