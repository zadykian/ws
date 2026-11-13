// Package sessiontest plays ws for the tests of a Session.
package sessiontest

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
)

// Ws is a Shell that plays ws on a test machine. The scripts' command, sh -s -- OP ARG..., runs
// on M as the command "op OP ARG...", so a test names the op alone and not the way to ws. Any other
// command runs on M unchanged.
type Ws struct{ M *effecttest.Machine }

// Op returns the Results key of the op with its args. Each word goes as SSHShell quotes it, so
// a word with a blank keys apart from the words it holds, as on ws.
func Op(op ...string) string {
	line := "op"
	for _, word := range op {
		line += " " + remote.Quote(word)
	}
	return line
}

// command returns the name and args that M runs for cmd. An op runs as its line, with no args.
func (f Ws) command(cmd []string) (string, []string) {
	if len(cmd) >= 4 && slices.Equal(cmd[:3], []string{"sh", "-s", "--"}) {
		return Op(cmd[3:]...), nil
	}
	return cmd[0], cmd[1:]
}

// Run runs cmd on M as the remote script would, with in on stdin.
func (f Ws) Run(ctx context.Context, in []byte, cmd ...string) ([]byte, error) {
	name, args := f.command(cmd)
	return f.M.RunInput(ctx, in, name, args...)
}

// Pass runs cmd on M and passes its output lines on.
func (f Ws) Pass(ctx context.Context, in []byte, cmd ...string) error {
	name, args := f.command(cmd)
	return f.M.PassInput(ctx, in, name, args...)
}

// Lines runs cmd on M and offers each output line to take.
func (f Ws) Lines(ctx context.Context, in []byte, take func(line string) bool, cmd ...string) error {
	name, args := f.command(cmd)
	return f.M.PassLines(ctx, in, take, name, args...)
}

// Reach is a session.Reach whose shell is Ws on M. The Reach it holds answers the rest, and may be
// nil where a test reaches nothing else.
type Reach struct {
	session.Reach
	M *effecttest.Machine
}

// Shell is Ws on M.
func (r Reach) Shell() remote.Shell { return Ws{M: r.M} }

// On returns a session over Ws on m, whose remote script answers status with the value's text.
func On(m *effecttest.Machine, status session.Status) *session.Session {
	return over(Reach{M: m}, m, status)
}

// New returns ws's own session, whose host_vars names ada, and whose remote script answers status
// with the value's text.
func New(status session.Status) (*session.Session, *effecttest.Machine) {
	m := effecttest.NewMachine()
	want := func() (session.Want, error) { return session.Want{Target: "lan", User: "ada"}, nil }
	reach := session.WsReach{M: m, Read: want, Here: m, Out: &effecttest.Out{}}
	return over(Reach{Reach: reach, M: m}, m, status), m
}

func over(r session.Reach, m *effecttest.Machine, status session.Status) *session.Session {
	ws := session.New(r, session.Scripts{Remote: []byte("x"), Files: []byte("y")})
	m.Results[Op("status")] = effecttest.Result{Out: Text(status)}
	return ws
}

// Text is the status as the remote script prints it, with - for an empty field, so that a stage's
// test states ws's status as a value.
func Text(s session.Status) []byte {
	word := func(set bool, yes, no string) string {
		if set {
			return yes
		}
		return no
	}
	run := "-"
	if s.Run != (session.Stamp{}) {
		run = strings.Join([]string{cmp.Or(s.Run.Version, "-"), cmp.Or(s.Run.Exit, "-"), word(s.Run.Off, "off", "on"),
			cmp.Or(s.Run.Secrets, "-"), cmp.Or(s.Run.Vars, "-")}, " ")
	}
	lines := []string{"run " + run, "fans " + word(s.FansRefused, "refused", "ok"), "peer " + word(s.Peered, "ok", "-"),
		"outbound " + word(s.OutboundOff, "off", "on")}
	for name, value := range map[string]string{"active": s.Active, "ws": s.Version, "user": s.User,
		"mac.pub": s.MacKey, "wg0": s.WgKey, "signing": s.Signing, "hostname": s.Hostname, "claude": s.Claude,
		"cable": s.Cable, "secrets": s.Secrets, "vars": s.Vars} {
		lines = append(lines, name+" "+cmp.Or(value, "-"))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}
