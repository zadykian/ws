package claude

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/systemd"
	"github.com/zadykian/ws/ubuntu/roles/claude/internal/cld"
	"github.com/zadykian/ws/ubuntu/values"
)

func TestRestorePartsStartTheAccountsSystemdFirst(t *testing.T) {
	a := account.Account{User: account.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}
	parts := RestoreParts(values.Values{}, a, effecttest.NewMachine(), &effecttest.Units{}, &effecttest.Out{})
	if len(parts) != 2 {
		t.Fatalf("parts = %v, want the user's systemd and cld's restore", parts)
	}
	if u, ok := parts[0].(systemd.Unit); !ok || u.Name != "user@1000.service" || u.Run != systemd.Running {
		t.Errorf("first part = %#v, want user@1000.service running", parts[0])
	}
	if r, ok := parts[1].(cld.Restore); !ok || r.Account != a {
		t.Errorf("second part = %#v, want cld's restore for the account", parts[1])
	}
}

func TestMarketplacesPartsOfAnEmptyListAreNone(t *testing.T) {
	var v values.Values
	if parts := MarketplacesParts(v, account.Account{}, effecttest.NewMachine(), nil, nil); parts != nil {
		t.Errorf("parts = %v, want none", parts)
	}
	v.Claude.Marketplaces = []values.Marketplace{acme}
	if parts := MarketplacesParts(v, account.Account{}, effecttest.NewMachine(), nil, nil); len(parts) != 1 {
		t.Errorf("parts = %v, want the marketplaces", parts)
	}
}
