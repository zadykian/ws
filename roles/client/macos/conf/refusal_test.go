package conf

import "testing"

func TestQuestionRefusal(t *testing.T) {
	need := Question{Key: "WS_REMOTE", Need: "WS_TARGET=cloud needs WS_REMOTE"}
	if got := need.Refusal("", ""); got != need.Need {
		t.Errorf("empty = %q", got)
	}
	if got := (Question{Key: "WS_REMOTE"}).Refusal("", ""); got != "" {
		t.Errorf("empty without a need = %q", got)
	}
	if got := (Question{Key: "WS_REMOTE_PORT"}).Refusal("99999", ""); got != "WS_REMOTE_PORT=99999 is not a port, 1 to 65535" {
		t.Errorf("bad port = %q", got)
	}
	addr := Question{Key: "WS_LAN_ADDRESS"}
	if got := addr.Refusal("10.88.0.10", "10.88.0.0/24"); got != "" {
		t.Errorf("address in its LAN = %q", got)
	}
	if got := addr.Refusal("10.9.0.10", "10.88.0.0/24"); got == "" {
		t.Error("address outside its LAN taken")
	}
}
