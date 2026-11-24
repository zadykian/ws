package secrets

import (
	"cmp"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// Recipients reads the recipients of a fnox file's age provider by their labels.
func Recipients(text []byte, provider string) (map[string]string, error) {
	conf, err := parseConfig(text, provider)
	if err != nil {
		return nil, err
	}
	return conf.Recipients, nil
}

// The forms of a line that WithRecipients writes. A label is a TOML bare key. A recipient is an age
// recipient or a host key, and neither needs an escape in a TOML string.
var (
	recipientLabel = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	recipientValue = regexp.MustCompile(`^(age1[0-9a-z]+|ssh-ed25519 [A-Za-z0-9+/]+={0,2})$`)
	recipientLine  = regexp.MustCompile(`^\s*[A-Za-z0-9_-]+\s*=\s*"[^"\\]*"\s*$`)
)

// WithRecipients is text with the table [providers.PROVIDER.recipients] holding r alone: nano,
// then backup, then the other labels sorted. It rewrites the table's lines and keeps every other
// byte, as a TOML encoder would drop fnox's comments and reorder its tables. A table that holds a
// line of another form stops it, as a rewrite would lose that line.
func WithRecipients(text []byte, provider string, r map[string]string) ([]byte, error) {
	for label, recipient := range r {
		if !recipientLabel.MatchString(label) {
			return nil, fmt.Errorf("the label %q is no TOML bare key of letters, digits, _ and -", label)
		}
		if !recipientValue.MatchString(recipient) {
			return nil, fmt.Errorf("the recipient of %s is neither age1... nor ssh-ed25519 BASE64", label)
		}
	}
	header := regexp.MustCompile(`^\s*\[\s*providers\s*\.\s*` + regexp.QuoteMeta(provider) +
		`\s*\.\s*recipients\s*\]\s*(#.*)?$`)
	lines := strings.SplitAfter(string(text), "\n")
	start := slices.IndexFunc(lines, func(l string) bool { return header.MatchString(strings.TrimRight(l, "\n")) })
	if start < 0 {
		return nil, fmt.Errorf("no table [providers.%s.recipients] of its own lines", provider)
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "[") {
			end = i
			break
		}
	}
	// The table's blank lines before its first entry and after its last stay where they are.
	body := lines[start+1 : end]
	first, last := len(body), -1
	for i, l := range body {
		switch {
		case strings.TrimSpace(l) == "":
		case recipientLine.MatchString(strings.TrimRight(l, "\n")):
			first, last = min(first, i), i
		default:
			return nil, fmt.Errorf("the table [providers.%s.recipients] holds a line other than "+
				"label = \"recipient\", which ws would lose as it rewrites the table: %q", provider, strings.TrimSpace(l))
		}
	}
	var b strings.Builder
	b.WriteString(strings.Join(lines[:start+1], ""))
	if !strings.HasSuffix(lines[start], "\n") {
		b.WriteString("\n")
	}
	if last < 0 {
		// A table of no entry keeps its blank lines after the new ones.
		first = 0
	}
	b.WriteString(strings.Join(body[:first], ""))
	for _, label := range labelOrder(r) {
		fmt.Fprintf(&b, "%s = \"%s\"\n", label, r[label])
	}
	b.WriteString(strings.Join(body[last+1:], ""))
	b.WriteString(strings.Join(lines[end:], ""))

	out := []byte(b.String())
	back, err := Recipients(out, provider)
	if err != nil {
		return nil, fmt.Errorf("the rewritten table [providers.%s.recipients] reads back as no fnox file: %w", provider, err)
	}
	if !maps.Equal(back, r) {
		return nil, fmt.Errorf("the rewritten table [providers.%s.recipients] reads back as other recipients: "+
			"the file sets them elsewhere too", provider)
	}
	return out, nil
}

// labelOrder is the order of a table's labels: the Nano's, the backup's, then the servers' by name.
func labelOrder(r map[string]string) []string {
	rank := func(label string) int {
		switch label {
		case NanoLabel:
			return 0
		case BackupLabel:
			return 1
		}
		return 2
	}
	return slices.SortedFunc(maps.Keys(r), func(a, b string) int {
		return cmp.Or(cmp.Compare(rank(a), rank(b)), cmp.Compare(a, b))
	})
}
