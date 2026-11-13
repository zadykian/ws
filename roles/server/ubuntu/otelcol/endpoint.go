// Package otelcol is the otelcol role as Go steps. They set up node_exporter, and the
// OpenTelemetry Collector with its configuration. The collector feeds SigNoz and, where
// ~/.secrets/env names one, a remote collector for claude's metrics. See docs/otelcol.md.
package otelcol

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// variable is the setting of ~/.secrets/env that names the remote collector.
const variable = "OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT"

var assignment = regexp.MustCompile(`^[[:space:]]*(?:export[[:space:]]+)?` + variable + `=(.*)$`)

// Endpoint returns the value of the variable in the text of an env file. It takes the last line
// that sets it, as the shell does, with one pair of quotes removed. The file is read, not sourced, so that
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

var (
	urlForm = regexp.MustCompile(`^https?://([A-Za-z0-9._-]+|\[[0-9A-Fa-f:.]+\]):([1-9][0-9]{0,4})$`)
	// receiver is the URL of the collector's own gRPC receiver, on 127.0.0.1, an unspecified
	// address, either one as IPv6, and localhost or a name under it.
	receiver = regexp.MustCompile(`(?i)^https?://(127\.0\.0\.1|0\.0\.0\.0|\[[0:]*\]|` +
		`\[[0:]*:ffff:(127\.0\.0\.1|0\.0\.0\.0)\]|([^/]*\.)?localhost\.?):4317$`)
)

// errEndpoint leaves the value out, as the value is a secret.
var errEndpoint = errors.New("OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT in ~/.secrets/env is neither " +
	"http://HOST:PORT nor https://HOST:PORT, with PORT from 1 to 65535, or it leads to the " +
	"collector's own receiver")

// CheckEndpoint refuses a URL that is no http or https URL with a host and a port. It also refuses
// one that leads to the collector's own gRPC receiver, which would send claude's metrics back to
// both pipelines without end. The collector checks the URL only as it starts. The error leaves the value out.
func CheckEndpoint(url string) error {
	m := urlForm.FindStringSubmatch(url)
	if m == nil {
		return errEndpoint
	}
	if port, _ := strconv.Atoi(m[2]); port > 65535 || receiver.MatchString(url) {
		return errEndpoint
	}
	return nil
}
