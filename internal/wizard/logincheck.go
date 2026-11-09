package wizard

import (
	"context"
	"regexp"
	"strings"
)

// loginStage names the stage in its messages.
const loginStage = "Login"

// Login is the stage that checks that WS_USER on ws takes the Mac's key, so that ssh logs in to ws.
// The wizard adds no key to ws. ws's keys come from the GitHub account at its install, and the gh
// on the Mac stage has put the Mac's there. So the stage stops where WS_USER refuses the key, and
// says what to do.
type Login struct {
	// Host is the Host key stage, whose ssh, address and known_hosts this one uses.
	Host *HostKey
	// Gh names the Mac's key in the stop.
	Gh GhCLI
}

// Name is the stage's name.
func (*Login) Name() string { return loginStage }

// WaitsOn is the Host key stage, without which ssh logs in nowhere.
func (*Login) WaitsOn() string { return hostStage }

// exitStatus is what the machine's error holds before the stderr of a command.
var exitStatus = regexp.MustCompile(`^.*exit status \d+: `)

// login tells whether ssh logs in to ws at its direct way as WS_USER by the Mac's key, and runs a
// command. It connects only where known_hosts holds a key under ws. ssh's error comes back in one line, as ssh
// ends each line with \r\n and an agent's refusal to sign comes before its last line.
func (r *Login) login(ctx context.Context, w Way) (ok bool, sshErr string) {
	if !r.Host.known(ctx, w.KnownHosts) {
		return false, ""
	}
	out, err := r.Host.SSH.Run(ctx, "-o", "BatchMode=yes", "ws", "echo", "ok")
	if err == nil && strings.TrimSpace(string(out)) == "ok" {
		return true, ""
	}
	if err == nil {
		return false, ""
	}
	return false, stderrOf(err)
}

// stderrOf returns the stderr that a failed command's error holds, in one line, without its argv.
func stderrOf(err error) string {
	var lines []string
	for _, l := range strings.Split(exitStatus.ReplaceAllString(err.Error(), ""), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return strings.Join(lines, "; ")
}
