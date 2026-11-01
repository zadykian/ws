//go:build !darwin

package main

// groups returns the command groups that exist on this system: none here, as ws mac is built for
// darwin alone.
func groups() []group { return nil }
