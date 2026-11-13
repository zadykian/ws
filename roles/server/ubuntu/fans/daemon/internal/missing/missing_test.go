package missing

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsSeesAnErrorThroughAWrap(t *testing.T) {
	err := fmt.Errorf("fan 1: %w", Errorf("no %s", "hwmon"))
	if !Is(err) || err.Error() != "fan 1: no hwmon" {
		t.Errorf("Is(%q) = %v, want true", err, Is(err))
	}
	if Is(errors.New("busy")) || Is(nil) {
		t.Error("Is is true for an error that is not missing")
	}
}
