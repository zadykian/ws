package wizard

import "testing"

func TestQuestionPrompt(t *testing.T) {
	for _, tc := range []struct {
		q    question
		want string
	}{
		{question{"WS_TARGET", "The machine", "lan", "needs"}, "The machine (WS_TARGET) [lan]"},
		{question{"WS_LAN", "ws's home LAN", "10.88.0.0/24", ""}, "ws's home LAN (WS_LAN) [10.88.0.0/24, or none]"},
		{question{"WS_RESOLVERS", "Another", "", ""}, "Another (WS_RESOLVERS) [none]"},
		{question{"WS_REMOTE", "The VM", "", "needs"}, "The VM (WS_REMOTE)"},
	} {
		if got := tc.q.prompt(); got != tc.want {
			t.Errorf("prompt = %q, want %q", got, tc.want)
		}
	}
}

func TestQuestionReply(t *testing.T) {
	q := question{def: "lan"}
	for raw, want := range map[string]string{"": "lan", "none": "", `"a b"`: "a b", "'x'": "x", "cloud": "cloud", `"`: `"`} {
		if got := q.reply(raw); got != want {
			t.Errorf("reply(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestHasLine(t *testing.T) {
	if !hasLine("# c\n  WS_LAN=1\n", "WS_LAN") || hasLine("X_WS_LAN=1\nWS_LAN_ADDRESS=2\n", "WS_LAN") {
		t.Error("hasLine takes a line of another key, or misses its own")
	}
}
