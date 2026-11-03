package values

// Base is roles/base's keys.
type Base struct {
	// Packages are the apt packages the role installs.
	Packages []string `yaml:"base_packages"`
	// Locale is the system locale.
	Locale string `yaml:"base_locale"`
	// Timezone is the system's time zone.
	Timezone string `yaml:"base_timezone"`
	// SwapSizeMB is the swap file's size in MiB, and 0 is none. Nil, unset, is the RAM rounded up to a
	// whole GiB.
	SwapSizeMB *int `yaml:"base_swap_size_mb,omitempty"`
	// Swappiness is vm.swappiness.
	Swappiness int `yaml:"base_swappiness"`
}

// Claude is roles/claude's keys.
type Claude struct {
	// Permissions is the permission set that cld adds to claude's user settings: read-only, cld
	// or none.
	Permissions string `yaml:"claude_permissions"`
	// Notifications is the channel claude notifies on. Empty sets none.
	Notifications string `yaml:"claude_notifications"`
	// TelemetryEnv is the environment that sends claude's telemetry to the collector.
	TelemetryEnv map[string]string `yaml:"claude_telemetry_env"`
	// Marketplaces are the plugin marketplaces the role adds for the user.
	Marketplaces []Marketplace `yaml:"claude_marketplaces"`
}

// Marketplace is a plugin marketplace: the name claude gives it and where it is from.
type Marketplace struct {
	Name   string `yaml:"name,omitempty"`
	Source string `yaml:"source,omitempty"`
}

// Devtools is roles/devtools' keys.
type Devtools struct {
	// ChromiumLibraries are the apt packages that a headless Chromium needs.
	ChromiumLibraries []string `yaml:"devtools_chromium_libraries"`
}

// Docker is roles/docker's keys. A string left empty, and the quota and memory limits, follow
// the machine: memory, CPUs and the disk of Docker's root.
type Docker struct {
	Packages           []string `yaml:"docker_packages"`
	LogMaxSize         string   `yaml:"docker_log_max_size"`
	LogMaxFile         int      `yaml:"docker_log_max_file"`
	MemoryMax          string   `yaml:"docker_memory_max"`
	CPUQuota           string   `yaml:"docker_cpu_quota"`
	MemorySwapMax      string   `yaml:"docker_memory_swap_max"`
	BridgeIP           string   `yaml:"docker_bridge_ip"`
	BridgePrefixLength int      `yaml:"docker_bridge_prefix_length"`
	AddressPool        string   `yaml:"docker_address_pool"`
	AddressPoolSize    int      `yaml:"docker_address_pool_size"`
	PublishIP          string   `yaml:"docker_publish_ip"`
	BuildCacheKeepDays int      `yaml:"docker_build_cache_keep_days"`
	// The build cache's limits, each in the form 12GB. Empty, each is a share of the disk.
	BuildCacheReservedSpace string `yaml:"docker_build_cache_reserved_space"`
	BuildCacheMaxUsedSpace  string `yaml:"docker_build_cache_max_used_space"`
	BuildCacheMinFreeSpace  string `yaml:"docker_build_cache_min_free_space"`
}
