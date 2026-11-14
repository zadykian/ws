package conf

import (
	"strings"
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

func TestRefuseBadSiblingNamesTheFileThatIsRefused(t *testing.T) {
	m := machineWith("ws", "lab", "box")
	m.Files[ServerPath(macPath, "lab")] = []byte("WS_GIT_NAME=Ada\n")
	err := RefuseBadSibling(m, macPath, ServerPath(macPath, "ws"))
	if err == nil || !strings.HasPrefix(err.Error(), "server lab: ") || !strings.HasSuffix(err.Error(), BadFileHint("lab")) {
		t.Errorf("RefuseBadSibling = %v, want lab's refusal", err)
	}
}

func TestRefuseBadSiblingLeavesTheServerItselfAlone(t *testing.T) {
	m := machineWith("ws", "box")
	m.Files[ServerPath(macPath, "ws")] = []byte("WS_GIT_NAME=Ada\n")
	if err := RefuseBadSibling(m, macPath, ServerPath(macPath, "ws")); err != nil {
		t.Errorf("RefuseBadSibling = %v, want none for the server's own file", err)
	}
}
