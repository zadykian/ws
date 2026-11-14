package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// twoServers is ws and lab on a Mac.
func twoServers() *effecttest.Machine {
	m := effecttest.NewMachine()
	keyed(m)
	m.Files["/h/.config/ws/servers/lab/server.conf"] = []byte("WS_USER=ada\nWS_SLOT=1\n")
	m.Dirs["/h/.config/ws/servers/lab"] = 0o755
	return m
}

func TestMacStateGroupsSeveralServersUnderTheirNames(t *testing.T) {
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = twoServers()
	status, stdout, stderr := runWs(p, "", "client", "macos", "state", "--plain")
	if status != 0 || !strings.Contains(stdout, "lab\n  ") || !strings.Contains(stdout, "ws\n  ") ||
		strings.Index(stdout, "lab\n") > strings.Index(stdout, "ws\n") {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want a section for lab, then ws", status, stdout, stderr)
	}
}

func TestMacStateJSONListsServersWhereThereAreSeveral(t *testing.T) {
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = twoServers()
	status, stdout, _ := runWs(p, "", "client", "macos", "state", "--json")
	var doc struct {
		Steps   []json.RawMessage `json:"steps"`
		Servers []struct {
			Name  string
			Steps []json.RawMessage
		}
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil || status != 0 {
		t.Fatalf("status = %d, err = %v, stdout = %q", status, err, stdout)
	}
	if len(doc.Steps) != 0 || len(doc.Servers) != 2 || doc.Servers[0].Name != "lab" || len(doc.Servers[1].Steps) == 0 {
		t.Errorf("doc = %+v; want servers lab and ws, and no steps at the top", doc)
	}
}

func TestMacStateOfOneServerKeepsItsBytes(t *testing.T) {
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = twoServers()
	status, stdout, _ := runWs(p, "", "client", "macos", "state", "--json", "--server", "lab")
	if status != 0 || strings.Contains(stdout, `"servers"`) || !strings.Contains(stdout, `"steps"`) {
		t.Errorf("status = %d, stdout = %q; want steps alone", status, stdout)
	}
	status, stdout, _ = runWs(p, "", "client", "macos", "state", "--plain", "--server", "lab")
	if status != 0 || strings.HasPrefix(stdout, "lab\n") {
		t.Errorf("status = %d, stdout = %q; want no header with --server", status, stdout)
	}
}

// refusedLab is two servers where lab's slot is out of range.
func refusedLab() *effecttest.Machine {
	m := twoServers()
	m.Files["/h/.config/ws/servers/lab/server.conf"] = []byte("WS_USER=ada\nWS_SLOT=99\n")
	return m
}

func TestMacStateKeepsTheGoodServersBesideARefusedFile(t *testing.T) {
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = refusedLab()
	status, stdout, stderr := runWs(p, "", "client", "macos", "state", "--plain")
	if status != 0 || !strings.Contains(stdout, "lab\n  server.conf\n") ||
		!strings.Contains(stdout, "fix its file, or ws client macos remove lab") ||
		!strings.Contains(stdout, "ws\n  ") {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want lab's refusal and ws's section", status, stdout, stderr)
	}
}

func TestMacStateJSONCarriesARefusedFile(t *testing.T) {
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = refusedLab()
	status, stdout, _ := runWs(p, "", "client", "macos", "state", "--json")
	var doc struct {
		Servers []struct {
			Name  string
			Steps []json.RawMessage
		}
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil || status != 0 || len(doc.Servers) != 2 ||
		!strings.Contains(stdout, "ws client macos remove lab") || len(doc.Servers[1].Steps) < 2 {
		t.Errorf("status = %d, err = %v, stdout = %q; want both servers, lab refused", status, err, stdout)
	}
}
