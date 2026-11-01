package wizard

import (
	"errors"
	"reflect"
	"slices"
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

func TestSequenceWays(t *testing.T) {
	for name, tc := range map[string]struct {
		answers map[string]string
		want    []string
	}{
		"cloud":  {map[string]string{"WS_TARGET": "cloud"}, []string{"WS_TARGET", "WS_REMOTE", "WS_REMOTE_PORT"}},
		"mac-vm": {map[string]string{"WS_TARGET": "mac-vm", "WS_LAN": "192.168.64.0/24"}, []string{"WS_TARGET", "WS_LAN", "WS_LAN_ADDRESS"}},
		"no lan": {map[string]string{"WS_TARGET": "mac-vm"}, []string{"WS_TARGET", "WS_LAN"}},
		"lan":    {map[string]string{"WS_TARGET": "lan"}, []string{"WS_TARGET", "WS_LAN", "WS_REMOTE"}},
	} {
		var asked []string
		if err := newSequence(nil, scripted(&asked, tc.answers)).ways(); err != nil || !reflect.DeepEqual(asked, tc.want) {
			t.Errorf("%s: asked %v, %v; want %v", name, asked, err, tc.want)
		}
	}
}

func TestSequenceRun(t *testing.T) {
	var asked []string
	answers := map[string]string{"WS_TARGET": "cloud", "WS_GIT_NAME": "Ada", "WS_HOOK": "me/hook@v1.0.0"}
	if err := newSequence(map[string]bool{"WS_TUNNEL_NETWORKS": true}, scripted(&asked, answers)).run(); err != nil {
		t.Fatal(err)
	}
	want := []string{"WS_TARGET", "WS_REMOTE", "WS_REMOTE_PORT", "WS_GIT_NAME", "WS_GIT_EMAIL", "WS_RESOLVERS", "WS_HOOK", "WS_HOOK_SECRET"}
	if !reflect.DeepEqual(asked, want) {
		t.Errorf("asked %v, want %v", asked, want)
	}
	asked = nil
	if err := newSequence(nil, scripted(&asked, nil)).run(); err != nil || slices.Contains(asked, "WS_GIT_EMAIL") || slices.Contains(asked, "WS_HOOK_SECRET") {
		t.Errorf("without a name or a hook: asked %v, %v", asked, err)
	}
}
