package values

// Base is roles/base's keys.
type Base struct {
	// Locale is the system locale.
	Locale string `yaml:"WS_BASE_LOCALE"`
	// Timezone is the system's time zone.
	Timezone string `yaml:"WS_BASE_TIMEZONE"`
	// SwapSizeMB is the swap file's size in MiB, and 0 is none. Nil, unset, is the RAM rounded up to a
	// whole GiB.
	SwapSizeMB *int `yaml:"WS_BASE_SWAP_SIZE_MB,omitempty"`
	// Swappiness is vm.swappiness.
	Swappiness int `yaml:"WS_BASE_SWAPPINESS"`
}

// Claude is roles/claude's keys.
type Claude struct {
	// Permissions is the permission set that cld adds to claude's user settings: read-only, cld
	// or none.
	Permissions string `yaml:"WS_CLAUDE_PERMISSIONS"`
	// Notifications is the channel claude notifies on. Empty sets none.
	Notifications string `yaml:"WS_CLAUDE_NOTIFICATIONS"`
	// TelemetryEnv is the environment that sends claude's telemetry to the collector.
	TelemetryEnv map[string]string `yaml:"WS_CLAUDE_TELEMETRY_ENV"`
	// Marketplaces are the plugin marketplaces the role adds for the user.
	Marketplaces []Marketplace `yaml:"WS_CLAUDE_MARKETPLACES"`
}

// Marketplace is a plugin marketplace: the name claude gives it and where it comes from.
type Marketplace struct {
	Name   string `yaml:"name,omitempty"`
	Source string `yaml:"source,omitempty"`
}

// Docker is roles/docker's keys. A string left empty, and the quota and memory limits, follow
// the machine: memory, CPUs and the disk of Docker's root.
type Docker struct {
	LogMaxSize         string `yaml:"WS_DOCKER_LOG_MAX_SIZE"`
	LogMaxFile         int    `yaml:"WS_DOCKER_LOG_MAX_FILE"`
	MemoryMax          string `yaml:"WS_DOCKER_MEMORY_MAX"`
	CPUQuota           string `yaml:"WS_DOCKER_CPU_QUOTA"`
	MemorySwapMax      string `yaml:"WS_DOCKER_MEMORY_SWAP_MAX"`
	BridgeIP           string `yaml:"WS_DOCKER_BRIDGE_IP"`
	BridgePrefixLength int    `yaml:"WS_DOCKER_BRIDGE_PREFIX_LENGTH"`
	AddressPool        string `yaml:"WS_DOCKER_ADDRESS_POOL"`
	AddressPoolSize    int    `yaml:"WS_DOCKER_ADDRESS_POOL_SIZE"`
	PublishIP          string `yaml:"WS_DOCKER_PUBLISH_IP"`
	BuildCacheKeepDays int    `yaml:"WS_DOCKER_BUILD_CACHE_KEEP_DAYS"`
	// The build cache's limits, each in the form 12GB. Empty, each is a share of the disk.
	BuildCacheReservedSpace string `yaml:"WS_DOCKER_BUILD_CACHE_RESERVED_SPACE"`
	BuildCacheMaxUsedSpace  string `yaml:"WS_DOCKER_BUILD_CACHE_MAX_USED_SPACE"`
	BuildCacheMinFreeSpace  string `yaml:"WS_DOCKER_BUILD_CACHE_MIN_FREE_SPACE"`
}
