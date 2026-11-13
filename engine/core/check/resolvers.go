package check

import "strings"

// Resolvers reports whether list is DOMAIN=ADDRESS[,ADDRESS...] split by blanks, with 1 to 3 IPv4
// addresses each. Each domain comes once and in lower case, as /etc/resolver's volume ignores case
// by default: two spellings would share one file.
func Resolvers(list string) bool {
	if strings.Trim(list, "abcdefghijklmnopqrstuvwxyz0123456789.,=- \t") != "" {
		return false
	}
	seen := map[string]bool{}
	for _, pair := range strings.Fields(list) {
		domain, addresses, found := strings.Cut(pair, "=")
		if !found || !Name(domain) || seen[domain] {
			return false
		}
		seen[domain] = true
		parts := strings.Split(addresses, ",")
		if len(parts) > 3 {
			return false
		}
		for _, a := range parts {
			if !IPv4(a) {
				return false
			}
		}
	}
	return true
}
