package devtools

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

// fingerprints returns the fingerprint of each certificate's primary key in an armored public key,
// in upper-case hex. Each public key packet starts a certificate, and every packet must belong to
// one. So the list names every key apt would trust from the file, and a subkey belongs to its
// certificate. It reads packet by packet, so a packet it cannot parse fails the check, where
// openpgp.ReadKeyRing would skip it. Primary keys of version 4 and 6 pass, no other.
func fingerprints(armored string) ([]string, error) {
	data, err := dearmor(armored)
	if err != nil {
		return nil, err
	}
	var prints []string
	r := bytes.NewReader(data)
	for {
		p, err := packet.Read(r)
		if errors.Is(err, io.EOF) {
			return prints, nil
		}
		if err != nil {
			return nil, fmt.Errorf("a packet: %w", err)
		}
		switch p := p.(type) {
		case *packet.PublicKey:
			if p.IsSubkey {
				if len(prints) == 0 {
					return nil, errors.New("a subkey in no certificate")
				}
				continue
			}
			if p.Version != 4 && p.Version != 6 {
				return nil, fmt.Errorf("a version %d key", p.Version)
			}
			prints = append(prints, strings.ToUpper(hex.EncodeToString(p.Fingerprint)))
		case *packet.Signature, *packet.UserId, *packet.UserAttribute, *packet.Marker, packet.Padding:
			if len(prints) == 0 {
				return nil, fmt.Errorf("a %T packet in no certificate", p)
			}
		default:
			return nil, fmt.Errorf("a %T packet in a public key", p)
		}
	}
}
