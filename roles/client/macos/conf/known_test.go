package conf

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestKnownAddsTheCurrentServerWhereItIsMissing(t *testing.T) {
	got, err := Known(machineWith("lab", "box"), macPath, "ws")
	if err != nil || strings.Join(got, ",") != "box,lab,ws" {
		t.Errorf("Known = %v, %v; want box, lab, ws", got, err)
	}
}

func TestKnownListsTheCurrentServerOnce(t *testing.T) {
	got, err := Known(machineWith("lab", "ws"), macPath, "ws")
	if err != nil || strings.Join(got, ",") != "lab,ws" {
		t.Errorf("Known = %v, %v; want lab, ws", got, err)
	}
}

func TestKnownGivesTheCurrentServerWithNoServersDirectory(t *testing.T) {
	got, err := Known(effecttest.NewMachine(), macPath, "ws")
	if err != nil || strings.Join(got, ",") != "ws" {
		t.Errorf("Known = %v, %v; want ws", got, err)
	}
}
