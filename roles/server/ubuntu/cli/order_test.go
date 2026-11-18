package cli

import (
	"slices"
	"testing"
)

// The collector's units run after the shell's env file, which holds the endpoint they read.
func TestOtelcolRunsAfterTheEnv(t *testing.T) {
	root, account := rootAndAccount(t)
	for _, names := range [][]string{specLines(t, root), specLines(t, account)} {
		env, deb := slices.Index(names, "file/shell-env"), slices.Index(names, "deb/otelcol")
		if env < 0 || deb < 0 || env > deb {
			t.Errorf("file/shell-env at %d, deb/otelcol at %d; want the env first", env, deb)
		}
		if svc := slices.Index(names, "service/otelcol"); svc < deb {
			t.Errorf("service/otelcol at %d before deb/otelcol at %d", svc, deb)
		}
	}
}
