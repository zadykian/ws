package kind

import (
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/swap"
)

// supportedLocales is glibc's list of the locales it can generate, as the locales package ships it.
const supportedLocales = "/usr/share/i18n/SUPPORTED"

func TestLocaleStateGeneratesAMissingLocale(t *testing.T) {
	e, m, _, _ := testEnv()
	m.Files[supportedLocales] = []byte("en_US.UTF-8 UTF-8\nde_DE.UTF-8 UTF-8\n")
	m.Files["/etc/locale.gen"] = []byte("# en_US.UTF-8 UTF-8\n")
	m.Results["locale -a"] = effecttest.Result{Out: []byte("C\n")}
	units := loadState(t, e, "locale = { name = \"en_US.UTF-8\", hint = \"Set X\" }\n")
	got := headlines(t, units[0].Step)
	want := []string{"write /etc/locale.gen", "generate the locale with locale-gen", "create /etc/locale.conf"}
	if !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q", got, want)
	}
	m.Files["/etc/locale.gen"] = []byte("en_US.UTF-8 UTF-8\n")
	m.Files["/etc/locale.conf"] = []byte("LANG=en_US.UTF-8\n")
	m.Results["locale -a"] = effecttest.Result{Out: []byte("C\nen_US.utf8\n")}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("listed: plan = %q, want none", got)
	}
}

func TestLocaleStateNamesItsHint(t *testing.T) {
	e, m, _, _ := testEnv()
	m.Files[supportedLocales] = []byte("en_US.UTF-8 UTF-8\n")
	units := loadState(t, e, "locale = { name = \"xx_YY.UTF-8\", hint = \"Set WS_X to one it lists\" }\n")
	_, err := units[0].Step.Plan(t.Context())
	if err == nil || !strings.HasSuffix(err.Error(), ". Set WS_X to one it lists") {
		t.Errorf("plan = %v, want a refusal that ends with the hint", err)
	}
	stateRefused(t, e, "locale = { hint = \"h\" }\n", "a locale state takes a name")
}

func TestTimezoneStateRefusesAPathOut(t *testing.T) {
	e, _, _, _ := testEnv()
	for _, name := range []string{"../x", "/x", "", "Etc/../x", ".."} {
		stateRefused(t, e, "timezone = { name = \""+name+"\", hint = \"Set WS_X\" }\n",
			"is no name below /usr/share/zoneinfo/, such as Etc/UTC. Set WS_X")
	}
}

func TestTimezoneStateLinksTheZone(t *testing.T) {
	e, m, _, _ := testEnv()
	m.Files["/usr/share/zoneinfo/Europe/Paris"] = []byte("tz")
	m.Links["/etc/localtime"] = "/usr/share/zoneinfo/Etc/UTC"
	units := loadState(t, e, "timezone = { name = \"Europe/Paris\", hint = \"h\" }\n")
	got := headlines(t, units[0].Step)
	if len(got) != 1 || !strings.Contains(got[0], "/etc/localtime") {
		t.Fatalf("plan = %q, want the link alone", got)
	}
	m.Links["/etc/localtime"] = "/usr/share/zoneinfo/Europe/Paris"
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("linked: plan = %q, want none", got)
	}
	if !slices.Contains(m.Calls, "stat /etc/timezone") {
		t.Errorf("calls %q, want /etc/timezone looked for", m.Calls)
	}
}

func TestSwapStateTakesTheRAMWithoutASize(t *testing.T) {
	e, _, _, _ := testEnv()
	// The swap's plan reads the disk of the machine that runs it, so the test reads the resource
	// alone: its size, nil for the RAM's, and its hint.
	for text, want := range map[string]*int{
		"swap = { hint = \"h\" }\n":                nil,
		"swap = { size_mib = 0, hint = \"h\" }\n":  new(0),
		"swap = { size_mib = 64, hint = \"h\" }\n": new(64),
	} {
		var s stateSpec
		if err := toml.Unmarshal([]byte(text), &s); err != nil {
			t.Fatal(err)
		}
		r, err := s.Swap.resource(e)
		got, ok := r.(swap.File)
		if err != nil || !ok || got.Hint != "h" || got.M != e.M || (got.SizeMiB == nil) != (want == nil) ||
			want != nil && *got.SizeMiB != *want {
			t.Errorf("%q: resource = %+v, %v; want a swap file of %v MiB", text, r, err, want)
		}
	}
}

func TestSwapStateRefusesANegativeSize(t *testing.T) {
	e, _, _, _ := testEnv()
	stateRefused(t, e, "swap = { size_mib = -1, hint = \"Set WS_X\" }\n", "swap: size_mib -1 is no size: it takes a whole number of MiB, or 0 for none. Set WS_X")
}
