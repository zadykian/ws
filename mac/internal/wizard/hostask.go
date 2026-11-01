package wizard

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/mac/internal/check"
)

func (*HostKey) none(w Way) string {
	return fmt.Sprintf("ws doesn't answer at %s:%s with an ed25519 key", w.Host, w.Port)
}

// ask asks ws's address now, where ws doesn't answer at its way with an ed25519 key, as a new ws
// with a lease from the router's pool. WS_FIRST_ADDRESS gives it, else the answer at the terminal,
// where none stops the run, as before Ubuntu's install. The question names where ws's machine shows
// it (docs/mac-targets.md). ssh takes it as HostName for the rest of the run, from the lock's file
// that the later stages read, and the stage reads ws's key there.
func (h *HostKey) ask(ctx context.Context, target string, w Way) (Way, string, error) {
	none := h.none(w)
	first, set := h.Env("WS_FIRST_ADDRESS")
	if !set {
		if !h.Tty {
			return w, "", fmt.Errorf("%s: %s, and no terminal is there to ask its address now: set WS_FIRST_ADDRESS",
				hostStage, none)
		}
		h.Out.Say("%s: %s; where Ubuntu isn't installed on ws yet, an empty answer stops the run", hostStage, none)
		where := "as the router's leases show it"
		switch target {
		case "cloud":
			where = "as the provider shows the VM's public one"
		case "mac-vm":
			where = "as ip -4 address prints it at ws's console"
		}
		var err error
		first, err = h.P.Ask(ctx, effect.Question{Text: "ws's address now, " + where, Valid: func(a string) error {
			if a == "" || check.IPv4(a) {
				return nil
			}
			return fmt.Errorf("%s: %s is not an IPv4 address", hostStage, a)
		}})
		if err != nil {
			return w, "", fmt.Errorf("%s: the terminal ended before ws's address", hostStage)
		}
		if first == "" {
			return w, "", fmt.Errorf("%s: no address for ws; once Ubuntu is installed on it, run the wizard again", hostStage)
		}
	}
	if !check.IPv4(first) {
		return w, "", fmt.Errorf("%s: WS_FIRST_ADDRESS=%s is not an IPv4 address", hostStage, first)
	}
	if err := h.M.WriteFile(h.Lock+"/"+FirstFile, []byte(first+"\n"), 0o600); err != nil {
		return w, "", err
	}
	either := fmt.Errorf("%s: ws doesn't answer at %s:%s with an ed25519 key either", hostStage, first, w.Port)
	nw, err := h.SSH.Way(ctx)
	if err != nil {
		return w, "", either
	}
	if line := h.scan(ctx, nw); line != "" {
		return nw, line, nil
	}
	return w, "", either
}
