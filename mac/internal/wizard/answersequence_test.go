package wizard

import (
	"errors"
	"reflect"
	"testing"
)

// scripted asks what a script says for each key, and records the keys asked.
func scripted(asked *[]string, answers map[string]string) func(question) (string, error) {
	return func(q question) (string, error) {
		*asked = append(*asked, q.key)
		return answers[q.key], nil
	}
}

func newSequence(lines map[string]bool, ask func(question) (string, error)) *sequence {
	return &sequence{vals: map[string]string{}, lacks: func(k string) bool { return !lines[k] }, ask: ask,
		role: func(name string) string { return "role " + name }}
}

func TestSequenceGetKeepsMacConf(t *testing.T) {
	var asked []string
	s := newSequence(map[string]bool{"WS_LAN": true}, scripted(&asked, nil))
	s.vals["WS_LAN"] = "10.1.0.0/24"
	if v, err := s.get(question{key: "WS_LAN"}); v != "10.1.0.0/24" || err != nil || len(asked) != 0 {
		t.Errorf("get = %q, %v, asked %v; want mac.conf's value, nothing asked", v, err, asked)
	}
	s.ask = func(question) (string, error) { return "", errors.New("stop") }
	if _, err := s.get(question{key: "WS_REMOTE"}); err == nil || s.vals["WS_REMOTE"] != "" {
		t.Errorf("get with a failing ask = %v", err)
	}
}

func TestSequenceLanWays(t *testing.T) {
	var asked []string
	s := newSequence(nil, scripted(&asked, map[string]string{"WS_LAN": "10.88.0.0/24", "WS_REMOTE": "r.example"}))
	if err := s.lanWays(); err != nil {
		t.Fatal(err)
	}
	if want := []string{"WS_LAN", "WS_LAN_ADDRESS", "WS_REMOTE", "WS_REMOTE_PORT"}; !reflect.DeepEqual(asked, want) {
		t.Errorf("asked %v, want %v", asked, want)
	}
	asked = nil
	if err := newSequence(nil, scripted(&asked, nil)).lanWays(); err != nil || !reflect.DeepEqual(asked, []string{"WS_LAN", "WS_REMOTE"}) {
		t.Errorf("without a LAN or a router: asked %v, %v", asked, err)
	}
}
