package cli

// secretInputs names each of mise's secret inputs that a role's mise.secrets.toml declares. An
// input's name is its variable's too, which the secrets task's fnox sets in the run's env. A run
// state drops each from the env of a command that is not ws's (kind.Env.Hidden), so that an
// installer the run fetches never sees a value. TestSecretInputsAreTheHiddenList holds the list
// to the files.
var secretInputs = []string{}
