package otelcol

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const packageQuery = "dpkg-query -W -f ${Package} ${Version} ${db:Status-Abbrev}\n otelcol-contrib"

func machine(have string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results[packageQuery] = effecttest.Result{Out: []byte(strings.TrimSpace("otelcol-contrib "+have) + "\n")}
	return m
}

func TestTheURLIsTheReleasesDeb(t *testing.T) {
	want := "https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/" +
		"v" + version + "/otelcol-contrib_" + version + "_linux_arm64.deb"
	if got := debURL(version, "arm64"); got != want {
		t.Errorf("got %s", got)
	}
}
