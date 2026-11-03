package signoz

import (
	"crypto/rand"
	"math/big"
)

// alphabet is the letters and digits of the password.
const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// passwordLength is the count of random letters and digits.
const passwordLength = 32

// passwordSuffix makes sure the password has the symbol, upper-case letter and digit that SigNoz
// requires.
const passwordSuffix = "-Aa1"

func randomPassword() (string, error) {
	out := make([]byte, passwordLength)
	for i := range out {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out), nil
}
