package values

// Default returns each key's default value, with the keys that follow ws_home or the machine's
// size left empty (see derive and each key).
func Default() Values {
	return Values{
		Shared: Shared{User: "zadykian"},
		Base: Base{
			Packages: []string{
				"btop", "bubblewrap", "build-essential", "ca-certificates", "chrony", "curl",
				"fd-find", "git", "gnupg", "gzip", "htop", "iotop", "jq", "locales", "lsb-release",
				"ncdu", "pkg-config", "poppler-utils", "python3-pip", "python3-venv", "ripgrep",
				"tar", "tzdata", "unattended-upgrades", "unzip", "wget", "xz-utils", "zip",
			},
			Locale: "en_US.UTF-8", Timezone: "Etc/UTC", Swappiness: 10,
		},
		Claude: Claude{
			Permissions: "read-only",
			TelemetryEnv: map[string]string{
				"CLAUDE_CODE_ENABLE_TELEMETRY":        "1",
				"OTEL_METRICS_EXPORTER":               "otlp",
				"OTEL_EXPORTER_OTLP_PROTOCOL":         "grpc",
				"OTEL_EXPORTER_OTLP_ENDPOINT":         "http://127.0.0.1:4317",
				"OTEL_TRACES_EXPORTER":                "otlp",
				"OTEL_LOGS_EXPORTER":                  "otlp",
				"CLAUDE_CODE_ENHANCED_TELEMETRY_BETA": "1",
				"OTEL_LOG_TOOL_DETAILS":               "1",
				"OTEL_LOG_USER_PROMPTS":               "1",
				"OTEL_LOG_ASSISTANT_RESPONSES":        "1",
				"OTEL_LOG_TOOL_CONTENT":               "1",
			},
		},
		Devtools: Devtools{ChromiumLibraries: []string{
			"libasound2t64", "libatk-bridge2.0-0t64", "libatk1.0-0t64", "libcairo2", "libcups2t64",
			"libgbm1", "libnspr4", "libnss3", "libpango-1.0-0", "libxcomposite1", "libxdamage1",
			"libxfixes3", "libxkbcommon0", "libxrandr2",
		}},
		Docker: Docker{
			Packages: []string{
				"docker-ce", "docker-ce-cli", "containerd.io", "docker-buildx-plugin",
				"docker-compose-plugin",
			},
			LogMaxSize: "10m", LogMaxFile: 3, MemorySwapMax: "0",
			BridgeIP: "172.17.0.1", BridgePrefixLength: 16,
			AddressPool: "172.16.0.0/12", AddressPoolSize: 20,
			PublishIP: "127.0.0.1", BuildCacheKeepDays: 90,
		},
		Fans:     defaultFans(),
		Helix:    Helix{Grammars: defaultGrammars()},
		Ikeward:  Ikeward{},
		Network:  Network{DirectLinkAddress: "10.77.0.1/30"},
		Security: Security{Outbound: true},
		Shell: Shell{
			GitName: "zadykian", GitEmail: "me@zadykian.dev", GitSigning: true,
		},
		Signoz: Signoz{
			Dir: "/opt/signoz", URL: "http://127.0.0.1:3301", RetentionDays: 90,
			Dashboards: []string{"host", "claude"},
		},
		Wireguard: Wireguard{
			Address: "10.99.0.1/24", Port: 51820,
			Peers: []WireguardPeer{{Name: "mac", Address: "10.99.0.2"}},
		},
		Zed: Zed{Nofile: 65536},
	}
}

func defaultGrammars() []string {
	return []string{
		"go", "gomod", "gotmpl", "gowork", "c-sharp", "xml", "bash", "diff", "json", "markdown",
		"markdown_inline", "toml", "yaml",
	}
}
