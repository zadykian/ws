package conf

import (
	"testing"
)

func TestSiblingsReadTheOtherServersByName(t *testing.T) {
	m := machineWith("ws", "lab", "box")
	m.Files[ServerPath(macPath, "lab")] = []byte("WS_RESOLVERS=a.test=10.0.0.1\n")
	got, err := Siblings(m, macPath, ServerPath(macPath, "ws"))
	if err != nil || len(got) != 2 || got[0].Name != "box" || got[1].Name != "lab" {
		t.Fatalf("Siblings = %+v, %v; want box and lab", got, err)
	}
	if got[1].File != ServerPath(macPath, "lab") || got[1].Conf.Resolvers != "a.test=10.0.0.1" {
		t.Errorf("lab = %+v", got[1])
	}
}

func TestSiblingsLeaveOutAFileThatIsRefused(t *testing.T) {
	m := machineWith("ws", "lab", "box")
	m.Files[ServerPath(macPath, "lab")] = []byte("WS_GIT_NAME=Ada\n")
	got, err := Siblings(m, macPath, ServerPath(macPath, "ws"))
	if err != nil || len(got) != 1 || got[0].Name != "box" {
		t.Errorf("Siblings = %+v, %v; want box alone", got, err)
	}
}

func TestSiblingsOfTheOnlyServerAreNone(t *testing.T) {
	got, err := Siblings(machineWith("ws"), macPath, ServerPath(macPath, "ws"))
	if err != nil || len(got) != 0 {
		t.Errorf("Siblings = %+v, %v; want none", got, err)
	}
}
