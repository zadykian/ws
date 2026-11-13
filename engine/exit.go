// Package engine runs a side's work for the commands: the errors that decide ws's exit status,
// and the run of a Target.
package engine

import "syscall"

// UsageError is an error of the command line: a wrong command, flag or count of arguments.
type UsageError struct{ Err error }

func (e UsageError) Error() string { return e.Err.Error() }

func (e UsageError) Unwrap() error { return e.Err }

// SignalExit is the error of a run that a signal cancelled. ws exits with 128 plus the signal's
// number, as a shell reports it.
type SignalExit struct{ Sig syscall.Signal }

func (e SignalExit) Error() string { return e.Sig.String() }
