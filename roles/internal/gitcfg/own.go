package gitcfg

import (
	"regexp"
	"slices"
	"strings"
)

var ownSetting = regexp.MustCompile(`^(user\.(name|email|signingkey)|(commit|tag)\.gpgsign|` +
	`gpg\.(format|ssh\.program|ssh\.allowedsignersfile))$`)

// Own returns the settings of identity and signing among names, once each, in their order, joined
// by ", ". names are those that git config --name-only --list prints.
func Own(names []string) string {
	var list []string
	for _, n := range names {
		if ownSetting.MatchString(n) && !slices.Contains(list, n) {
			list = append(list, n)
		}
	}
	return strings.Join(list, ", ")
}
