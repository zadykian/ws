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

// generateLine makes an identity on the Nano, through Pass.
const generateLine = secrets.Plugin + " --generate --name ws-secrets --pin-policy once --touch-policy always"

// emptyKey is the list of a YubiKey that holds no age identity: the blank line that
// age-plugin-yubikey prints after each YubiKey it reads.
const emptyKey = "\n"

// listing is a Mac whose YubiKey lists out.
func listing(out string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Results[listLine] = effecttest.Result{Out: []byte(out)}
	return m
}

func TestNanoTakesTheSlotOfFnoxsRecipient(t *testing.T) {
	m := listing(twoSlots)
	s, err := secrets.NanoIdentity(context.Background(), m, &strings.Builder{}, recipientA)
	if err != nil || s != (secrets.Slot{Slot: 5, Recipient: recipientA}) {
		t.Errorf("NanoIdentity gives %+v, %v, want slot 5", s, err)
	}
}

func TestNanoRefusesAKeyWithoutFnoxsRecipient(t *testing.T) {
	for _, list := range []string{twoSlots, emptyKey} {
		m := listing(list)
		_, err := secrets.NanoIdentity(context.Background(), m, &strings.Builder{}, "age1yubikey1qc")
		if err == nil || !strings.Contains(err.Error(), "age1yubikey1qc") {
			t.Errorf("a YubiKey without fnox.toml's recipient gives %v, want an error naming it", err)
		}
		if slices.Contains(m.Calls, "run "+generateLine) {
			t.Errorf("a YubiKey without fnox.toml's recipient made one: %q", m.Calls)
		}
	}
}

func TestNanoTakesTheFirstSlotWhereFnoxNamesNone(t *testing.T) {
	m := listing(twoSlots)
	var out strings.Builder
	s, err := secrets.NanoIdentity(context.Background(), m, &out, "")
	if err != nil || s != (secrets.Slot{Slot: 2, Recipient: recipientB}) {
		t.Errorf("NanoIdentity gives %+v, %v, want slot 2", s, err)
	}
	if !strings.Contains(out.String(), "slot 2") {
		t.Errorf("NanoIdentity says %q, want the slot it takes", out.String())
	}
}

func TestNanoGeneratesWhereTheKeyHoldsNone(t *testing.T) {
	m := listing(emptyKey)
	m.Results[generateLine] = effecttest.Result{}
	m.After = map[string]map[string]effecttest.Result{generateLine: {listLine: {Out: []byte(twoSlots)}}}
	s, err := secrets.NanoIdentity(context.Background(), m, &strings.Builder{}, "")
	if err != nil || s.Slot != 2 {
		t.Errorf("NanoIdentity gives %+v, %v, want the made slot 2", s, err)
	}
	want := []string{"run " + listLine, "run " + generateLine, "run " + listLine}
	if !slices.Equal(m.Calls, want) {
		t.Errorf("NanoIdentity ran\n%q\nwant\n%q", m.Calls, want)
	}
}

func TestNanoStopsWhereGenerateLeavesNone(t *testing.T) {
	m := listing(emptyKey)
	m.Results[generateLine] = effecttest.Result{}
	if _, err := secrets.NanoIdentity(context.Background(), m, &strings.Builder{}, ""); err == nil {
		t.Error("NanoIdentity takes a YubiKey that lists none after generate")
	}
	m.Results[generateLine] = effecttest.Result{Err: errors.New("exit status 1")}
	if _, err := secrets.NanoIdentity(context.Background(), m, &strings.Builder{}, ""); err == nil ||
		!strings.Contains(err.Error(), "cannot make") {
		t.Errorf("a failed generate gives %v, want an error", err)
	}
}

func TestNanoFindsNoYubiKeyWhereNoneIsPluggedIn(t *testing.T) {
	for _, want := range []string{"", recipientA} {
		m := listing("")
		m.Results[generateLine] = effecttest.Result{}
		_, err := secrets.NanoIdentity(context.Background(), m, &strings.Builder{}, want)
		if err == nil || !strings.Contains(err.Error(), "no YubiKey found") {
			t.Errorf("a list of nothing, with fnox.toml's recipient %q, gives %v, want no YubiKey found", want, err)
		}
		if slices.Contains(m.Calls, "run "+generateLine) {
			t.Errorf("a list of nothing made an identity: %q", m.Calls)
		}
	}
}
