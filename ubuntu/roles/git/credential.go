package git

import (
	"fmt"
	"io"

	"github.com/zadykian/ws/engine/core/effect"
)

// Answer is git's credential program for the URL whose key is key, as credentialKey gives it. git
// runs it as the account, with its action, and its request on the standard input. For get, it
// writes the URL's username and token from TokensFile. For store and erase it writes nothing, so that a
// token the server refuses stays as mise wrote it. Without the file, or the key in it, git gets
// nothing and asks its next program. It reads no values.yml, which the account may not read.
func Answer(m effect.Machine, key, action string, in io.Reader, out io.Writer) error {
	// git writes its request whole before it reads the answer: a program that leaves it unread may
	// break git's pipe.
	if _, err := io.Copy(io.Discard, in); err != nil {
		return err
	}
	if action != "get" {
		return nil
	}
	tokens, _, err := ReadTokens(m)
	if err != nil {
		return err
	}
	for _, c := range credentialsOf(tokens) {
		if credentialKey(c.URL) != key {
			continue
		}
		username := c.Username
		if username == "" {
			username = "token"
		}
		_, err := fmt.Fprintf(out, "username=%s\npassword=%s\n", username, c.Token)
		return err
	}
	return nil
}
