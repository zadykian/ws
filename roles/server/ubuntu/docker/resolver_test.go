package docker

import (
	"testing"

	"github.com/zadykian/ws/roles/server/ubuntu/internal/ufw"
)

func TestOwnsResolverRuleLeavesAlonePlainRules(t *testing.T) {
	own := ufw.Rule{Action: "allow", Proto: "any", ToPort: "53", FromPort: "any", Dir: "in", Interface: "docker0"}
	if !ownsResolverRule(own) {
		t.Fatal("a rule on an interface is not owned")
	}
	for name, mutate := range map[string]func(*ufw.Rule){
		"no interface": func(r *ufw.Rule) { r.Interface = "" },
		"a protocol":   func(r *ufw.Rule) { r.Proto = "udp" },
		"any port":     func(r *ufw.Rule) { r.ToPort = "any" },
	} {
		r := own
		mutate(&r)
		if ownsResolverRule(r) {
			t.Errorf("a rule with %s is owned", name)
		}
	}
}
