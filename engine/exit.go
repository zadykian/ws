// Package engine holds what the commands of both sides share: the errors that decide ws's exit
// status, and the lines a command prints.
package engine

import (
	"strconv"
	"syscall"
)

// UsageError is an error of the command line: a wrong command, flag or count of arguments.
type UsageError struct{ Err error }

func (e UsageError) Error() string { return e.Err.Error() }

func (e UsageError) Unwrap() error { return e.Err }

// SignalExit is the error of a run that a signal cancelled. ws exits with 128 plus the signal's
// number, as a shell reports it.
type SignalExit struct{ Sig syscall.Signal }

func (e SignalExit) Error() string { return e.Sig.String() }

// ExitStatus is the error of a command whose status is its answer, as a check's is to the program
// that runs it. ws exits with Code, and prints Err alone, or nothing without one.
type ExitStatus struct {
	Code int
	Err  error
}

func (e ExitStatus) Error() string {
	if e.Err == nil {
		return "exit status " + strconv.Itoa(e.Code)
	}
	return e.Err.Error()
}

func (e ExitStatus) Unwrap() error { return e.Err }
