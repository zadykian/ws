// Package sshkey holds the rule for the name of an SSH key of ws's, which both sides share. A name
// is a file name below ~/.ssh, so the rule keeps it from naming a path, a public key's file, or
// seward's socket and lock.
package sshkey

import (
	"errors"
	"fmt"
	"strings"
)

// reserved are the endings that a name may not have: the public key's file and seward's files.
var reserved = []string{".pub", ".sock", ".lock"}

// Check returns an error where name is no name of a key. A name holds letters, digits, '.', '_'
// and '-', and starts with a letter or a digit. It ends in none of .pub, .sock and .lock.
func Check(name string) error {
	if name == "" {
		return errors.New("the key's name is empty")
	}
	for i, r := range name {
		letter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
		if !letter && (i == 0 || !strings.ContainsRune("._-", r)) {
			return fmt.Errorf("the key's name %q holds %q at %d: use letters, digits, '.', '_' and '-', the first a letter or a digit", name, r, i)
		}
	}
	for _, end := range reserved {
		if strings.HasSuffix(name, end) {
			return fmt.Errorf("the key's name %q ends in %s, which ssh and seward keep for their own files", name, end)
		}
	}
	return nil
}
