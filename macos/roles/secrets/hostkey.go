package secrets

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
)

// hostKeyForm is a server's ed25519 host key as a recipient: the .pub's first two fields.
var hostKeyForm = regexp.MustCompile(`^ssh-ed25519 [A-Za-z0-9+/]+={0,2}$`)

// errNotSetUp stops a change to a repository that init has not set up.
var errNotSetUp = errors.New("the secrets repository is not set up: run ws macos secrets init")

// setHostKey makes key the recipient of hostname's label in fnox.toml and in hostname's profile
// file. It makes that file where missing, with the Nano's and the backup's recipients of
// fnox.toml. It takes key from any other label first, as a server whose hostname changed keeps
// its key. changed lists the files whose recipients changed, fnox.toml first, then by name: none
// where key was in place.
func setHostKey(f Files, hostname, key string) (out Files, changed []string, err error) {
	if err := CheckHostname(hostname); err != nil {
		return nil, nil, err
	}
	if !hostKeyForm.MatchString(key) {
		return nil, nil, errors.New("the host key is not of the form ssh-ed25519 BASE64")
	}
	text, ok := f[SharedFile]
	if !ok {
		return nil, nil, errNotSetUp
	}
	shared, err := Recipients(text, SharedProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", SharedFile, err)
	}
	if shared[NanoLabel] == "" || shared[BackupLabel] == "" {
		return nil, nil, errNotSetUp
	}
	set := func(r map[string]string) {
		dropRecipient(r, key)
		r[hostname] = key
	}
	e := edit{out: maps.Clone(f)}
	if err := e.apply(SharedFile, SharedProvider, set); err != nil {
		return nil, nil, err
	}
	own := ProfileFile(hostname)
	for _, name := range profileFiles(f) {
		change := func(r map[string]string) { dropRecipient(r, key) }
		if name == own {
			change = set
		}
		if err := e.apply(name, ProfileProvider, change); err != nil {
			return nil, nil, err
		}
	}
	if _, ok := f[own]; !ok {
		text, err := WithRecipients(fmt.Appendf(nil, profileStart, hostname, ProfileProvider), ProfileProvider,
			map[string]string{NanoLabel: shared[NanoLabel], BackupLabel: shared[BackupLabel], hostname: key})
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", own, err)
		}
		e.out[own] = text
		e.changed = append(e.changed, own)
	}
	return e.out, sharedFirst(e.changed), nil
}

// dropKey takes key from every table of recipients: fnox.toml's and each profile file's. A file
// stays, with its values, so that a rebuilt server takes its profile again. changed lists the
// files whose recipients changed, as setHostKey's does, and labels the labels that held key.
// Neither holds a name where key is no recipient.
func dropKey(f Files, key string) (out Files, changed, labels []string, err error) {
	if !hostKeyForm.MatchString(key) {
		return nil, nil, nil, errors.New("the host key is not of the form ssh-ed25519 BASE64")
	}
	held := map[string]bool{}
	drop := func(r map[string]string) {
		for label, recipient := range r {
			if recipient == key {
				held[label] = true
			}
		}
		dropRecipient(r, key)
	}
	e := edit{out: maps.Clone(f)}
	if _, ok := f[SharedFile]; ok {
		if err := e.apply(SharedFile, SharedProvider, drop); err != nil {
			return nil, nil, nil, err
		}
	}
	for _, name := range profileFiles(f) {
		if err := e.apply(name, ProfileProvider, drop); err != nil {
			return nil, nil, nil, err
		}
	}
	return e.out, sharedFirst(e.changed), slices.Sorted(maps.Keys(held)), nil
}

// profileStart is a new profile file, whose recipients WithRecipients sets.
const profileStart = `# The values of the server %[1]s alone, which fnox -P %[1]s set NAME --provider %[2]s adds.
[providers.%[2]s]
type = "age"

[providers.%[2]s.recipients]

[secrets]
`

// dropRecipient takes key from every label of r that holds it.
func dropRecipient(r map[string]string, key string) {
	maps.DeleteFunc(r, func(_, recipient string) bool { return recipient == key })
}

// profileFiles are the names of f's profile files, sorted.
func profileFiles(f Files) []string {
	var names []string
	for name := range f {
		if isProfileFile(name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// sharedFirst sorts names by name, with SharedFile first.
func sharedFirst(names []string) []string {
	slices.SortFunc(names, func(a, b string) int {
		switch {
		case a == b:
			return 0
		case a == SharedFile:
			return -1
		case b == SharedFile:
			return 1
		case a < b:
			return -1
		}
		return 1
	})
	return names
}

// edit rewrites the recipients of a repository's files, and keeps the names of those it changed.
type edit struct {
	out     Files
	changed []string
}

// apply changes the recipients of provider in the file name. It rewrites the file only where the
// recipients differ, as fnox's reencrypt rewrites every value of a file it runs on.
func (e *edit) apply(name, provider string, change func(map[string]string)) error {
	before, err := Recipients(e.out[name], provider)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	after := maps.Clone(before)
	change(after)
	if maps.Equal(before, after) {
		return nil
	}
	text, err := WithRecipients(e.out[name], provider, after)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	e.out[name] = text
	e.changed = append(e.changed, name)
	return nil
}
