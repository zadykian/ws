package devtools

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/devtools/internal/npm"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// Steps returns the role's steps: gh's telemetry and login, roslyn-language-server, the npm servers
// and rustup. The bootstrap's mise config installs the packages and the tools first.
func Steps(v values.Values, s secrets.Secrets, m effect.Machine, _ effect.Units, _ effect.Out) []step.Step {
	user := account.User{Name: v.User, Home: v.Home}
	return []step.Step{
		// gh's telemetry goes off before gh's login runs gh again.
		resource.Group{Title: "Devtools gh telemetry", Parts: []resource.Resource{
			GHTelemetry{M: m, User: user},
		}},
		// gh logs in by token.github, after its package.
		resource.Group{Title: "Devtools gh login", Parts: []resource.Resource{
			GHLogin{M: m, User: v.User, Home: v.Home, Token: s.Token.GitHub},
		}},
		resource.Group{Title: "Devtools roslyn-language-server", Parts: []resource.Resource{
			roslynTool{M: m, User: user, Path: Roslyn(v.Home)},
		}},
		user.WithAccount(m, "Devtools npm servers", "", func(a account.Account) ([]resource.Resource, error) {
			return npm.Files(m, a)
		}),
		resource.Group{Title: "Devtools rustup", Parts: []resource.Resource{
			Rustup{M: m, User: v.User, Home: v.Home},
		}},
	}
}
