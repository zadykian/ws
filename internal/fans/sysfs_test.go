package fans

import (
	"os"
	"strings"
	"testing"
)

func TestOSSysfsReads(t *testing.T) {
	path := t.TempDir() + "/pwm1"
	if err := os.WriteFile(path, []byte("128\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := (OSSysfs{}).Read(path); err != nil || got != "128" {
		t.Errorf("Read = %q, %v; want 128", got, err)
	}
}

func TestOSSysfsMissesAFileThatIsNotThere(t *testing.T) {
	_, err := OSSysfs{}.Read(t.TempDir() + "/none")
	if !IsMissing(err) || !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Read = %v, want a missing error that says no such file", err)
	}
}
