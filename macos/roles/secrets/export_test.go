package secrets

// The functions that HostKeys alone calls, for the tests of their own rules.
var (
	KnownKeys      = knownKeys
	SetHostKey     = setHostKey
	PlanDeployKeys = planDeployKeys
	AddDeployKey   = addDeployKey
	Strays         = strays
)

// Stray is stray, for the same tests.
type Stray = stray
