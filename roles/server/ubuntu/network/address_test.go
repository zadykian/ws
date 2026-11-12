package network

import (
	"slices"
	"testing"
)

func TestAddressProblems(t *testing.T) {
	tests := []struct {
		name, value string
		want        []string
	}{
		{"the default", "10.77.0.1/30", nil},
		{"the other host of a /30", "10.77.0.2/30", nil},
		{"a /31's first", "10.77.0.0/31", nil},
		{"a /31's second", "10.77.0.1/31", nil},
		{"a /32", "10.77.0.1/32", nil},
		{"a larger subnet's host", "10.77.1.1/24", nil},
		{"no prefix", "10.77.0.1", []string{"10.77.0.1 is not an IPv4 address with its prefix length, as 10.77.0.1/30"}},
		{"an empty prefix", "10.77.0.1/", []string{"10.77.0.1/ is not an IPv4 address with its prefix length, as 10.77.0.1/30"}},
		{"a netmask", "10.77.0.1/255.255.255.252", []string{"10.77.0.1/255.255.255.252 is not an IPv4 address with its prefix length, as 10.77.0.1/30"}},
		{"nothing", "", []string{" is not an IPv4 address with its prefix length, as 10.77.0.1/30"}},
		{"a prefix of 33", "10.77.0.1/33", []string{`10.77.0.1/33 is not an IPv4 address with its prefix length: "33" is not a prefix length of 0 to 32`}},
		{"an IPv6 address", "fd00::1/64", []string{`fd00::1/64 is not an IPv4 address with its prefix length: "fd00::1" is not an IPv4 address`}},
		{"a bad octet", "10.77.0.256/30", []string{`10.77.0.256/30 is not an IPv4 address with its prefix length: "10.77.0.256" is not an IPv4 address`}},
		{"the subnet's own address", "10.77.0.0/30", []string{"10.77.0.0 is the address of the subnet 10.77.0.0/30 itself, not of a host in it"}},
		{"the broadcast address", "10.77.0.3/30", []string{"10.77.0.3 is the broadcast address of 10.77.0.0/30, not a host's"}},
		{"a wider subnet's broadcast", "10.77.1.255/24", []string{"10.77.1.255 is the broadcast address of 10.77.1.0/24, not a host's"}},
		{"the whole space", "0.0.0.0/0", []string{"0.0.0.0 is the address of the subnet 0.0.0.0/0 itself, not of a host in it"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddressProblems(tt.value); !slices.Equal(got, tt.want) {
				t.Errorf("AddressProblems(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
