package keys

import (
	"strings"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const confPathForTest = "/h/.config/ws/mac.conf"

// callsWith returns the calls of m that hold word.
func callsWith(m *effecttest.Machine, word string) []string {
	var got []string
	for _, c := range m.Calls {
		if strings.Contains(c, word) {
			got = append(got, c)
		}
	}
	return got
}
