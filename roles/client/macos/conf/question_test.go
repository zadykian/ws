package conf

import "testing"

func TestQuestionPrompt(t *testing.T) {
	for _, tc := range []struct {
		q    Question
		want string
	}{
		{Question{"WS_TARGET", "The machine", "lan", "needs"}, "The machine (WS_TARGET) [lan]"},
		{Question{"WS_LAN", "ws's home LAN", "10.88.0.0/24", ""}, "ws's home LAN (WS_LAN) [10.88.0.0/24, or none]"},
		{Question{"WS_RESOLVERS", "Another", "", ""}, "Another (WS_RESOLVERS) [none]"},
		{Question{"WS_REMOTE", "The VM", "", "needs"}, "The VM (WS_REMOTE)"},
	} {
		if got := tc.q.Prompt(); got != tc.want {
			t.Errorf("prompt = %q, want %q", got, tc.want)
		}
	}
}

func TestQuestionReply(t *testing.T) {
	q := Question{Def: "lan"}
	for raw, want := range map[string]string{"": "lan", "none": "", `"a b"`: "a b", "'x'": "x", "cloud": "cloud", `"`: `"`} {
		if got := q.Reply(raw); got != want {
			t.Errorf("reply(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestHasLine(t *testing.T) {
	if !HasLine("# c\n  WS_LAN=1\n", "WS_LAN") || HasLine("X_WS_LAN=1\nWS_LAN_ADDRESS=2\n", "WS_LAN") {
		t.Error("HasLine takes a line of another key, or misses its own")
	}
}
