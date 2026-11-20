package secrets

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// Plugin is age-plugin-yubikey where state.age-plugin-yubikey copies it, on the PATH that fnox
// finds it by.
const Plugin = "/usr/local/bin/age-plugin-yubikey"

// stubPrefix starts the line of an identity stub: the YubiKey's serial, the slot and a tag of the
// recipient, which age-plugin-yubikey reads to find the key. It holds no secret.
const stubPrefix = "AGE-PLUGIN-YUBIKEY-1"

// Slot is an age identity that the YubiKey holds: its slot, of 1 to 20, and its recipient.
type Slot struct {
	Slot      int
	Recipient string
}

// slotLine is the comment that heads each identity of age-plugin-yubikey's list.
var slotLine = regexp.MustCompile(`^#\s*Serial: \d+, Slot: (\d+)$`)

// ParseList reads the identities of age-plugin-yubikey --list: each recipient line, with the slot
// of the comment above it. A YubiKey without one lists nothing.
func ParseList(out []byte) ([]Slot, error) {
	var slots []Slot
	slot := 0
	for line := range strings.Lines(string(out)) {
		line = strings.TrimSpace(line)
		if m := slotLine.FindStringSubmatch(line); m != nil {
			n, err := strconv.Atoi(m[1])
			if err != nil || n < 1 || n > 20 {
				return nil, fmt.Errorf("age-plugin-yubikey lists slot %s, not one of 1 to 20", m[1])
			}
			slot = n
			continue
		}
		if !strings.HasPrefix(line, "age1yubikey1") {
			continue
		}
		if slot == 0 {
			return nil, fmt.Errorf("age-plugin-yubikey lists the recipient %s under no slot", line)
		}
		slots = append(slots, Slot{Slot: slot, Recipient: line})
		slot = 0
	}
	return slots, nil
}

// ListNano lists the age identities of the YubiKey plugged in. age-plugin-yubikey prints a blank
// line after each YubiKey it reads, one without an identity too. Where none is plugged in, it
// prints nothing, and exits 0 too. So a list of nothing finds no YubiKey, not an empty one.
func ListNano(ctx context.Context, m effect.Machine) ([]Slot, error) {
	out, err := m.Run(ctx, Plugin, "--list")
	if err != nil {
		return nil, fmt.Errorf("cannot list the YubiKey's age identities: %w", err)
	}
	if len(out) == 0 {
		return nil, errors.New("no YubiKey found: plug in the Nano, and run it again")
	}
	return ParseList(out)
}

// NanoIdentity finds the Nano's age identity. Where fnox.toml names the Nano's recipient, want,
// it takes that recipient's slot alone: the values open to no other identity. Where fnox.toml
// names none, it takes the lowest slot listed, or one that age-plugin-yubikey makes on the
// terminal, with the Nano's PIN and a touch. It says which on out.
func NanoIdentity(ctx context.Context, m effect.Machine, out io.Writer, want string) (Slot, error) {
	slots, err := ListNano(ctx, m)
	if err != nil {
		return Slot{}, err
	}
	if want != "" {
		for _, s := range slots {
			if s.Recipient == want {
				_, _ = fmt.Fprintf(out, "the Nano holds fnox.toml's nano recipient in slot %d\n", s.Slot)
				return s, nil
			}
		}
		return Slot{}, fmt.Errorf("the YubiKey holds no identity of fnox.toml's nano recipient %s: plug in "+
			"the Nano that made it", want)
	}
	if len(slots) == 0 {
		_, _ = fmt.Fprintln(out, "the YubiKey holds no age identity: age-plugin-yubikey makes one, "+
			"which asks for the Nano's PIN and a touch")
		// Pass sends generate's output to ws's stderr, so a second list reads the new identity.
		if err := m.Pass(ctx, false, Plugin, "--generate", "--name", "ws-secrets", "--pin-policy", "once",
			"--touch-policy", "always"); err != nil {
			return Slot{}, fmt.Errorf("cannot make an age identity on the Nano: %w", err)
		}
		if slots, err = ListNano(ctx, m); err != nil {
			return Slot{}, err
		}
		if len(slots) == 0 {
			return Slot{}, errors.New("age-plugin-yubikey made an identity, yet the YubiKey lists none")
		}
	}
	s := slices.MinFunc(slots, func(a, b Slot) int { return a.Slot - b.Slot })
	_, _ = fmt.Fprintf(out, "the Nano's age identity is in slot %d: %s\n", s.Slot, s.Recipient)
	return s, nil
}

// Stub prints the identity file of the YubiKey's slot: comments, then the stub line, which fnox
// hands age-plugin-yubikey to decrypt. It needs no PIN and no touch.
func Stub(ctx context.Context, m effect.Machine, slot int) (string, error) {
	out, err := m.Run(ctx, Plugin, "--identity", "--slot", strconv.Itoa(slot))
	if err != nil {
		return "", fmt.Errorf("cannot read the identity of the YubiKey's slot %d: %w", slot, err)
	}
	if stubOf(string(out)) == "" {
		return "", fmt.Errorf("age-plugin-yubikey printed no %s line for the YubiKey's slot %d", stubPrefix, slot)
	}
	return string(out), nil
}

// stubOf returns the stub line of an identity file, or "" where it holds none.
func stubOf(text string) string {
	for line := range strings.Lines(text) {
		if line = strings.TrimSpace(line); strings.HasPrefix(line, stubPrefix) {
			return line
		}
	}
	return ""
}
