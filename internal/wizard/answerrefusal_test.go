package wizard

import "testing"

func TestQuestionRefusal(t *testing.T) {
	need := question{key: "WS_REMOTE", need: "WS_TARGET=cloud needs WS_REMOTE"}
	if got := need.refusal("", ""); got != need.need {
		t.Errorf("empty = %q", got)
	}
	if got := (question{key: "WS_REMOTE"}).refusal("", ""); got != "" {
		t.Errorf("empty without a need = %q", got)
	}
	if got := (question{key: "WS_REMOTE_PORT"}).refusal("99999", ""); got != "WS_REMOTE_PORT=99999 is not a port, 1 to 65535" {
		t.Errorf("bad port = %q", got)
	}
	addr := question{key: "WS_LAN_ADDRESS"}
	if got := addr.refusal("10.88.0.10", "10.88.0.0/24"); got != "" {
		t.Errorf("address in its LAN = %q", got)
	}
	if got := addr.refusal("10.9.0.10", "10.88.0.0/24"); got == "" {
		t.Error("address outside its LAN taken")
	}
}
