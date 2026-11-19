package secrets

import (
	"context"
	"fmt"
	"regexp"
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

// ListNano lists the age identities of the YubiKey plugged in.
func ListNano(ctx context.Context, m effect.Machine) ([]Slot, error) {
	out, err := m.Run(ctx, Plugin, "--list")
	if err != nil {
		return nil, fmt.Errorf("cannot list the YubiKey's age identities: %w", err)
	}
	return ParseList(out)
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
