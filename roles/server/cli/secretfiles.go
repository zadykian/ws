package cli

// secretInputs names each of mise's secret inputs that a role's mise.secrets.toml declares. An
// input's name is its variable's too, which the secrets task's fnox sets in the run's env.
// TestSecretInputsAreTheHiddenList holds the list to the files.
var secretInputs = []string{}
