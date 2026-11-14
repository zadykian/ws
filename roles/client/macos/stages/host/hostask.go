package host

import (
	"context"
	"fmt"
	"github.com/zadykian/ws/engine/core/remote"

	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// firstQuestion asks ws's address now, for the Host key stage and the Login stage, each from its
// own fields. ssh is the Mac's ssh to ws, whose lock takes the address.
type firstQuestion struct {
	ssh remote.SSH
	env func(string) (string, bool)
	tty bool
	p   effect.Prompt
	out effect.Out
}

// question is the Host key stage's.
func (h *Key) question() firstQuestion { return firstQuestion{h.SSH, h.Env, h.Tty, h.P, h.Out} }

func noAnswer(server string, w remote.Way) string {
	return fmt.Sprintf("%s doesn't answer at %s:%s with an ed25519 key", server, w.Host, w.Port)
}

// ask asks ws's address now, where ws doesn't answer at its way with an ed25519 key, as a new ws
// with a lease from the router's pool. WS_FIRST_ADDRESS gives it, else the answer at the terminal,
// where none stops the run, as before Ubuntu's install. The question names where ws's machine shows
// it (roles/client/macos/conf/README.md). ssh takes it as HostName for the rest of the run, from the lock's file
// that the later stages read, and the stage reads ws's key there.
func (q firstQuestion) ask(ctx context.Context, target string, w remote.Way) (remote.Way, string, error) {
	none := noAnswer(q.ssh.Host, w)
	first, set := q.env("WS_FIRST_ADDRESS")
	if !set {
		if !q.tty {
			return w, "", fmt.Errorf("%s: %s, and no terminal is there to ask its address now: set WS_FIRST_ADDRESS",
				stage.Host, none)
		}
		q.out.Say("%s: %s; where Ubuntu isn't installed on %s yet, an empty answer stops the run", stage.Host, none, q.ssh.Host)
		where := "as the router's leases show it"
		switch target {
		case "cloud":
			where = "as the provider shows the VM's public one"
		case "mac-vm":
			where = "as ip -4 address prints it at " + q.ssh.Host + "'s console"
		}
		var err error
		first, err = q.p.Ask(ctx, effect.Question{Text: q.ssh.Host + "'s address now, " + where, Valid: func(a string) error {
			if a == "" || check.IPv4(a) {
				return nil
			}
			return fmt.Errorf("%s: %s is not an IPv4 address", stage.Host, a)
		}})
		if err != nil {
			return w, "", fmt.Errorf("%s: the terminal ended before %s's address", stage.Host, q.ssh.Host)
		}
		if first == "" {
			return w, "", fmt.Errorf("%s: no address for %s; once Ubuntu is installed on it, run the wizard again", stage.Host, q.ssh.Host)
		}
	}
	if !check.IPv4(first) {
		return w, "", fmt.Errorf("%s: WS_FIRST_ADDRESS=%s is not an IPv4 address", stage.Host, first)
	}
	if err := q.ssh.M.WriteFile(q.ssh.Lock+"/"+remote.FirstFile, []byte(first+"\n"), 0o600); err != nil {
		return w, "", err
	}
	either := fmt.Errorf("%s: %s doesn't answer at %s:%s with an ed25519 key either", stage.Host, q.ssh.Host, first, w.Port)
	nw, err := q.ssh.Way(ctx)
	if err != nil {
		return w, "", either
	}
	if line := scanKey(ctx, q.ssh.M, nw); line != "" {
		return nw, line, nil
	}
	return w, "", either
}
