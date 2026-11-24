package values

import "reflect"

// HostKeys lists the keys that a line of the host's file holds, in the order of Host.
func HostKeys() []Key { return keysOf(reflect.TypeFor[Host]()) }

// Text is the key's value in h as a line gives it, which is empty for an unset pointer.
func (h Host) Text(k Key) string { return textOf(h, k) }

// YAML is the whole file for h, as Values.YAML gives the guest's.
func (h Host) YAML() (string, error) { return yamlOf(h) }

// SetHost is Set for the host's file.
func SetHost(data []byte, name, value string) ([]byte, error) {
	return setIn(data, name, value, HostKeys(), ParseHost, decodeHost)
}
