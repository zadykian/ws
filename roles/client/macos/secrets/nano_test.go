package secrets_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/secrets"
)

// The YubiKey's lines, as age-plugin-yubikey prints them. A stub here is short, as a real one
// would read as a key to a scan for secrets.
const (
	listLine     = secrets.Plugin + " --list"
	identityLine = secrets.Plugin + " --identity --slot 2"
	recipientA   = "age1yubikey1qa"
	recipientB   = "age1yubikey1qb"
	stubText     = "#    Recipient: " + recipientB + "\nAGE-PLUGIN-YUBIKEY-1QQQ\n"
)

// twoSlots is a list of two identities, in slots 2 and 5, in age-plugin-yubikey's format.
const twoSlots = `#       Serial: 12345678, Slot: 5
#         Name: other
#      Created: Thu, 19 Nov 2026 18:00:00 +0000
#   PIN policy: Once   (A PIN is required once per session, if set)
# Touch policy: Always (A physical touch is required for every decryption)
` + recipientA + `

#       Serial: 12345678, Slot: 2
#         Name: ws-secrets
#      Created: Thu, 19 Nov 2026 18:00:00 +0000
#   PIN policy: Once   (A PIN is required once per session, if set)
# Touch policy: Always (A physical touch is required for every decryption)
` + recipientB + `

`

func TestParseListTakesEachSlotsRecipient(t *testing.T) {
	slots, err := secrets.ParseList([]byte(twoSlots))
	want := []secrets.Slot{{Slot: 5, Recipient: recipientA}, {Slot: 2, Recipient: recipientB}}
	if err != nil || !slices.Equal(slots, want) {
		t.Errorf("ParseList gives %+v, %v, want %+v", slots, err, want)
	}
}

func TestParseListOfNoKeyIsEmpty(t *testing.T) {
	slots, err := secrets.ParseList(nil)
	if err != nil || len(slots) != 0 {
		t.Errorf("ParseList of nothing gives %+v, %v, want no slot", slots, err)
	}
}

func TestParseListRefusesARecipientUnderNoSlot(t *testing.T) {
	if _, err := secrets.ParseList([]byte(recipientA + "\n")); err == nil {
		t.Error("ParseList takes a recipient under no slot")
	}
}

func TestStubPrintsTheSlotsIdentityFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[identityLine] = effecttest.Result{Out: []byte(stubText)}
	stub, err := secrets.Stub(context.Background(), m, 2)
	if err != nil || stub != stubText {
		t.Errorf("Stub gives %q, %v, want the identity file", stub, err)
	}
}

func TestStubRefusesOutputWithoutAStub(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[identityLine] = effecttest.Result{Out: []byte("#    Recipient: " + recipientB + "\n")}
	if _, err := secrets.Stub(context.Background(), m, 2); err == nil ||
		!strings.Contains(err.Error(), "AGE-PLUGIN-YUBIKEY-1") {
		t.Errorf("Stub of no stub line gives %v, want an error naming the line", err)
	}
	m.Results[identityLine] = effecttest.Result{Err: errors.New("exit status 1")}
	if _, err := secrets.Stub(context.Background(), m, 2); err == nil || !strings.Contains(err.Error(), "slot 2") {
		t.Errorf("Stub of a failed run gives %v, want an error naming the slot", err)
	}
}
