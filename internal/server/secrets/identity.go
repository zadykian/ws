package secrets

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
	"golang.org/x/term"
)

// ReadIdentity reads the X25519 identity. At a terminal it asks on prompt and reads the line with
// no echo; else it reads all of stdin, as piped over ssh. Comment lines are skipped. The error
// names no content.
func ReadIdentity(stdin io.Reader, prompt io.Writer) (age.Identity, error) {
	text, err := readText(stdin, prompt)
	if err != nil {
		return nil, err
	}
	ids, err := age.ParseIdentities(strings.NewReader(text))
	if err != nil {
		return nil, errors.New("the age identity is not valid")
	}
	if len(ids) != 1 {
		return nil, fmt.Errorf("want one age identity, got %d", len(ids))
	}
	return ids[0], nil
}

func readText(stdin io.Reader, prompt io.Writer) (string, error) {
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		_, _ = io.WriteString(prompt, "age identity: ")
		line, err := term.ReadPassword(int(f.Fd()))
		_, _ = io.WriteString(prompt, "\n")
		return string(line), err
	}
	data, err := io.ReadAll(stdin)
	return string(data), err
}
