package values

// Default returns each key's default value, with the keys that derive fills or the machine's size
// left empty (see derive and each key).
func Default() Values {
	return Values{
		Base: Base{Locale: "en_US.UTF-8", Timezone: "Etc/UTC", Swappiness: 10},
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
		Docker: Docker{
			LogMaxSize: "10m", LogMaxFile: 3, MemorySwapMax: "0",
			BridgeIP: "172.17.0.1", BridgePrefixLength: 16,
			AddressPool: "172.16.0.0/12", AddressPoolSize: 20,
			PublishIP: "127.0.0.1", BuildCacheKeepDays: 90,
		},
		Fans:     defaultFans(),
		Ikeward:  Ikeward{},
		Cable:    Cable{Address: "10.77.0.1/30"},
		Security: Security{Outbound: true},
		Signoz: Signoz{
			URL: "http://127.0.0.1:3301", RetentionDays: 90,
			Dashboards: []string{"host", "claude"},
		},
		Tunnel: Tunnel{
			Address: "10.99.0.1/24", Port: 51820,
		},
		Zed: Zed{Nofile: 65536},
	}
}
