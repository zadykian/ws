package cli

import "github.com/zadykian/ws/roles/server/ubuntu/ikeward"

// secretInputs names each of mise's secret inputs that a role's mise.secrets.toml declares. An
// input's name is its variable's too, which the secrets task's fnox sets in the run's env. A run
// state drops each from the env of a command that is not ws's (kind.Env.Hidden), so that an
// installer the run fetches never sees a value. TestSecretInputsAreTheHiddenList holds the list
// to the files, and so ikeward's names, which ws reads, to its file.
var secretInputs = []string{"OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT", "WS_GITHUB_TOKEN", "WS_GIT_TOKENS",
	ikeward.P12Var, ikeward.SswanVar, ikeward.PasswordVar}

// secretsTask is the mise task that gives the inputs their values, mise-tasks/secrets of the side's
// run directory. ws reads an input from the run's env in its run alone (enginecli.Env.TaskEnv).
const secretsTask = "secrets"
