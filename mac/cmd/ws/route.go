package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/setup"
	"github.com/zadykian/ws/mac/internal/step"
)

const (
	routesUsage   = "usage: ws mac tunnel routes [--check] NAME [CIDR...]\n"
	resolverUsage = "usage: ws mac resolver [--check] DOMAIN [ADDRESS...]\n"
	loadedUsage   = "usage: ws mac tunnel loaded\n"
	daemonLabel   = "com.github.zadykian.ws-tunnel"
)

// tunnelLoaded succeeds where launchd has the tunnel's daemon loaded. Anyone may read launchd's
// system domain, so it needs no sudo.
func tunnelLoaded(ctx context.Context, args []string, m effect.Machine, stderr io.Writer) int {
	if len(args) != 0 {
		_, _ = io.WriteString(stderr, loadedUsage)
		return 2
	}
	if _, err := m.Run(ctx, "launchctl", "print", "system/"+daemonLabel); err != nil {
		return 1
	}
	return 0
}

// tunnelRoutes has the tunnel take the networks, or drop the file of NAME with none.
func tunnelRoutes(ctx context.Context, args []string, m effect.Machine, stderr io.Writer) int {
	check, args := checkFlag(args)
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, routesUsage)
		return 2
	}
	out, flush := routeOut(stderr)
	defer flush()
	return runRoute(ctx, &setup.Routes{Set: args[0], Networks: args[1:], M: m, Out: out, Check: check, Alone: true}, out, check, stderr)
}

// resolverCommand has macOS ask the nameservers for DOMAIN, or drop its file with none.
func resolverCommand(ctx context.Context, args []string, m effect.Machine, stderr io.Writer) int {
	check, args := checkFlag(args)
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, resolverUsage)
		return 2
	}
	out, flush := routeOut(stderr)
	defer flush()
	return runRoute(ctx, &setup.Resolver{Domain: args[0], Addresses: args[1:], M: m, Out: out}, out, check, stderr)
}

// checkFlag takes a leading --check off the arguments.
func checkFlag(args []string) (bool, []string) {
	if len(args) > 0 && args[0] == "--check" {
		return true, args[1:]
	}
	return false, args
}

// routeOut prints a line after "ws: ", as setup does, and returns the function that flushes it.
func routeOut(stderr io.Writer) (effect.Out, func()) {
	term := effect.NewTerminal(strings.NewReader(""), stderr)
	return progOut{term, "ws", map[string]bool{}}, term.Flush
}

// runRoute runs the step alone. Under --check it prints each change and makes none.
func runRoute(ctx context.Context, s step.Step, out effect.Out, check bool, stderr io.Writer) int {
	if _, err := step.Run(ctx, []step.Step{s}, out, check); err != nil {
		// The runner names the step and its stage before the step's own error.
		if inner := errors.Unwrap(err); inner != nil {
			err = inner
		}
		_, _ = fmt.Fprintf(stderr, "ws: %v\n", err)
		return 1
	}
	return 0
}
