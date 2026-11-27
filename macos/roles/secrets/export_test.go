package secrets

// The functions that HostKeys and Init alone call, for the tests of their own rules.
var (
	AddDeployKey    = addDeployKey
	DeleteDeployKey = deleteDeployKey
	DeployKeys      = deployKeys
	DropKey         = dropKey
	FnoxEnv         = fnoxEnv
	HasSecrets      = hasSecrets
	KnownKeys       = knownKeys
	PlanDeployKeys  = planDeployKeys
	ReadFiles       = readFiles
	Reencrypt       = reencrypt
	SetHostKey      = setHostKey
	Strays          = strays
	WriteFiles      = writeFiles
)

// Stray is stray, for the same tests.
type Stray = stray
