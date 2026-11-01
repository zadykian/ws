//go:build !darwin && !linux

package main

// groups returns the command groups that exist on this system: none here, as ws mac is built for
// darwin alone and ws server for Linux alone.
func groups() []group { return nil }
