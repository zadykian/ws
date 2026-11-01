package ufw_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/internal/server/ufw"
)

const userRules = `*filter
### RULES ###

### tuple ### allow any 53 172.17.0.1 any 0.0.0.0/0 in_docker0 comment=444e532066726f6d20636f6e7461696e657273
-A ufw-user-input -i docker0 -d 172.17.0.1 --dport 53 -j ACCEPT

### tuple ### allow tcp 22 0.0.0.0/0 any 0.0.0.0/0 in
### tuple ### allow any 80 0.0.0.0/0 any 0.0.0.0/0 in comment=zz
### tuple ### allow tcp 22 0.0.0.0/0 Apache any 0.0.0.0/0 in_eth0 comment=41
### tuple ### allow any 9 0.0.0.0/0 any 10.0.0.0/8 out comment=41
`

func TestParseReadsTheCommentedIPv4Tuples(t *testing.T) {
	want := []ufw.Rule{
		{Action: "allow", Proto: "any", ToPort: "53", To: "172.17.0.1", FromPort: "any", From: ufw.Any,
			Dir: "in", Interface: "docker0", Comment: "DNS from containers"},
		{Action: "allow", Proto: "any", ToPort: "9", To: ufw.Any, FromPort: "any", From: "10.0.0.0/8",
			Dir: "out", Comment: "A"},
	}
	if got := ufw.Parse(userRules); !slices.Equal(got, want) {
		t.Errorf("Parse = %+v, want %+v", got, want)
	}
}

func TestParseReadsARouteRuleAsRouteDoes(t *testing.T) {
	text := "### tuple ### route:allow any any 0.0.0.0/0 any 10.99.0.2 in_wg0!out_ikeward0 comment=696b6577617264\n"
	got := ufw.Parse(text)
	if want := []ufw.Rule{ufw.Route("allow", "wg0", "ikeward0", "10.99.0.2", "ikeward")}; !slices.Equal(got, want) {
		t.Errorf("Parse = %+v, want %+v", got, want)
	}
}
