package devtools

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/devtools/internal/golang"
	"github.com/zadykian/ws/roles/server/ubuntu/devtools/internal/npm"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// Steps returns the role's steps: gh's telemetry and login, Go, the language servers and rustup.
// The bootstrap's mise config installs the packages first.
func Steps(v values.Values, s secrets.Secrets, m effect.Machine, _ effect.Units, _ effect.Out) []step.Step {
	user := account.User{Name: v.User, Home: v.Home}
	steps := []step.Step{
		// gh's telemetry goes off before gh's login runs gh again.
		resource.Group{Title: "Devtools gh telemetry", Parts: []resource.Resource{
			GHTelemetry{M: m, User: user},
		}},
		// gh logs in by token.github, after its package.
		resource.Group{Title: "Devtools gh login", Parts: []resource.Resource{
			GHLogin{M: m, User: v.User, Home: v.Home, Token: s.Token.GitHub},
		}},
		resource.Group{Title: "Devtools Go", Parts: []resource.Resource{golang.Tree(m, GoVersion)}},
		resource.Group{Title: "Devtools gopls", After: "Devtools Go", Parts: []resource.Resource{
			golang.Gopls{M: m, User: user, Path: Gopls(v.Home), Version: goplsVersion, GoVersion: GoVersion},
		}},
		resource.Group{Title: "Devtools Delve", After: "Devtools Go", Parts: []resource.Resource{
			golang.Delve{M: m, User: user, Path: Dlv(v.Home), Version: dlvVersion, GoVersion: GoVersion},
		}},
		resource.Group{Title: "Devtools roslyn-language-server", Parts: []resource.Resource{
			roslynTool{M: m, User: user, Path: Roslyn(v.Home)},
		}},
	}
	return append(append(steps, serverSteps(m, user)...), resource.Group{Title: "Devtools rustup", Parts: []resource.Resource{
		Rustup{M: m, User: v.User, Home: v.Home},
	}})
}

// serverSteps returns the steps of the language servers that editors name by path. They are the
// downloads of the SQL and Docker servers, and the npm servers. They install
// apart from Steps so that a test can run them with no stand-ins for Go and the packages.
func serverSteps(m effect.Machine, user account.User) []step.Step {
	return []step.Step{
		download(m, "postgres-language-server", pgls, pglsURL),
		download(m, "docker-language-server", dockerLS, dockerLSURL),
		user.WithAccount(m, "Devtools npm servers", "", func(a account.Account) ([]resource.Resource, error) {
			return npm.Files(m, a)
		}),
	}
}
