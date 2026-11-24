package values

import (
	"strings"
	"testing"
)

const twoGuests = `WS_INCUS_GUESTS:
  - {name: lab, address: 10.100.0.10, cpu: 4, memory: 8GiB, disk: 200GiB}
  - {name: build-2, address: 10.100.0.11, cpu: 2, memory: 4096MiB, disk: 1TiB}
`

func TestNoGuestIsDeclaredByDefault(t *testing.T) {
	h, err := ParseHost(nil)
	if err != nil || len(h.Incus.Guests) != 0 {
		t.Errorf("guests = %+v, err = %v, want none: a host with no list runs none", h.Incus.Guests, err)
	}
}

func TestParseHostTakesTheGuests(t *testing.T) {
	h, err := ParseHost([]byte(twoGuests))
	if err != nil {
		t.Fatal(err)
	}
	want := Guest{Name: "build-2", Address: "10.100.0.11", CPU: 2, Memory: "4096MiB", Disk: "1TiB"}
	if len(h.Incus.Guests) != 2 || h.Incus.Guests[1] != want {
		t.Errorf("guests = %+v, want two, the second %+v", h.Incus.Guests, want)
	}
}

func TestParseHostRefusesABadGuest(t *testing.T) {
	good := "name: lab, address: 10.100.0.10, cpu: 4, memory: 8GiB, disk: 200GiB"
	for name, tc := range map[string]struct{ guest, want string }{
		"empty name":           {strings.Replace(good, "name: lab", "name: ''", 1), "name"},
		"a capital":            {strings.Replace(good, "lab", "Lab", 1), "name"},
		"a dot":                {strings.Replace(good, "lab", "a.b", 1), "name"},
		"a slash":              {strings.Replace(good, "lab", "a/b", 1), "name"},
		"a leading digit":      {strings.Replace(good, "lab", "1lab", 1), "name"},
		"a trailing dash":      {strings.Replace(good, "lab", "lab-", 1), "name"},
		"too long for its NIC": {strings.Replace(good, "lab", "abcdefghijklm", 1), "name"},
		"no address":           {strings.Replace(good, "address: 10.100.0.10", "address: ''", 1), "address"},
		"outside the subnet":   {strings.Replace(good, "10.100.0.10", "10.100.1.10", 1), "not in the bridge's subnet"},
		"the bridge's own":     {strings.Replace(good, "10.100.0.10", "10.100.0.1", 1), "the bridge's own"},
		"the broadcast":        {strings.Replace(good, "10.100.0.10", "10.100.0.255", 1), "not a host's address"},
		"no CPU":               {strings.Replace(good, "cpu: 4", "cpu: 0", 1), "cpu"},
		"a memory in bytes":    {strings.Replace(good, "8GiB", "8000", 1), "memory"},
		"a disk in percent":    {strings.Replace(good, "200GiB", "50%", 1), "disk"},
		"a zero disk":          {strings.Replace(good, "200GiB", "0GiB", 1), "disk"},
	} {
		data := "WS_INCUS_GUESTS:\n  - {" + tc.guest + "}\n"
		if _, err := ParseHost([]byte(data)); err == nil || !strings.Contains(err.Error(), "WS_INCUS_GUESTS") || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: ParseHost = %v, want WS_INCUS_GUESTS and %q", name, err, tc.want)
		}
	}
}

func TestParseHostRefusesTwoGuestsThatShareANameOrAnAddress(t *testing.T) {
	for name, data := range map[string]string{
		"a name":     strings.Replace(twoGuests, "build-2", "lab", 1),
		"an address": strings.Replace(twoGuests, "10.100.0.11", "10.100.0.10", 1),
	} {
		if _, err := ParseHost([]byte(data)); err == nil || !strings.Contains(err.Error(), "twice") {
			t.Errorf("two guests with the same %s: ParseHost = %v, want it said twice", name, err)
		}
	}
}
