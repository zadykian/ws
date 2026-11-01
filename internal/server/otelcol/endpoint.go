// Package otelcol is the otelcol role as Go steps: node_exporter, and the OpenTelemetry Collector
// with its configuration, which feeds SigNoz and, where ~/.secrets/env names one, a remote
// collector for claude's metrics. See docs/otelcol.md.
package otelcol

import (
	"regexp"
	"strings"
)

// variable is the setting of ~/.secrets/env that names the remote collector.
const variable = "OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT"

var assignment = regexp.MustCompile(`^[[:space:]]*(?:export[[:space:]]+)?` + variable + `=(.*)$`)

// Endpoint returns the value of the variable in the text of an env file: from the last line that
// sets it, as in the shell, with one pair of quotes removed. The file is read, not sourced, so that
// nothing in it runs as root. It returns "" where no line sets the variable.
func Endpoint(env []byte) string {
	value := ""
	for line := range strings.Lines(string(env)) {
		if m := assignment.FindStringSubmatch(strings.TrimRight(line, "\n")); m != nil {
			value = strings.TrimRight(m[1], " \t\n\v\f\r")
		}
	}
	if n := len(value); n >= 2 && (value[0] == '"' || value[0] == '\'') && value[n-1] == value[0] {
		value = value[1 : n-1]
	}
	return value
}
