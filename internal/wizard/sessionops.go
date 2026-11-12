package wizard

import (
	"context"
	"encoding/base64"
)

// operation is one job that a stage asks of ws through the session.
type operation string

const (
	opStatus    operation = "status"
	opState     operation = "state"
	opPorts     operation = "ports"
	opWait      operation = "wait"
	opNameCable operation = "name the cable"
	opNameUser  operation = "name the user"
	opNamePeer  operation = "name the peer"
	opInstall   operation = "install"
	opRun       operation = "run"
	opOutbound  operation = "outbound"
	opValues    operation = "values"
	opSetValues operation = "set values"
)

// outWay is what an op does with its output.
type outWay int

// outRead returns the output. outPass passes it to Stderr as it comes, for an op that goes on, such
// as install. outFollow is outPass for a run, whose events go to Events.
const (
	outRead outWay = iota
	outPass
	outFollow
)

// sent is one op of an operation: a word of a script's case table, and its output's way.
type sent struct {
	files bool
	word  string
	out   outWay
}

// opRule is how an operation runs: the ops it sends in turn, and whether it changes ws, so that
// the next status reads ws again.
type opRule struct {
	sends   []sent
	changes bool
}

// opRules is the one table of ws's operations. A word is one that the case table of its script
// holds, which TestOpsMatchTheScripts checks.
var opRules = map[operation]opRule{
	opStatus:    {sends: []sent{{word: "status"}}},
	opState:     {sends: []sent{{word: "state"}}},
	opPorts:     {sends: []sent{{files: true, word: "link"}}},
	opWait:      {sends: []sent{{files: true, word: "wait"}}},
	opNameCable: {changes: true, sends: []sent{{true, "link", outPass}, {false, "run", outFollow}}},
	opNameUser:  {changes: true, sends: []sent{{files: true, word: "user"}}},
	opNamePeer:  {changes: true, sends: []sent{{true, "peer", outPass}, {false, "run", outFollow}}},
	opInstall:   {changes: true, sends: []sent{{false, "install", outPass}}},
	opRun:       {changes: true, sends: []sent{{false, "run", outFollow}}},
	opOutbound:  {changes: true, sends: []sent{{word: "outbound"}}},
	opValues:    {sends: []sent{{files: true, word: "values"}}},
	opSetValues: {changes: true, sends: []sent{{files: true, word: "values"}}},
}

// do runs the operation: the ops of its row in turn, the args[i] after the word of the i-th, and
// stops at the first error. It marks the status stale before the first op where the row changes
// ws. It returns the output of the last op that reads.
func (s *Session) do(ctx context.Context, o operation, args ...[]string) ([]byte, error) {
	rule := opRules[o]
	if rule.changes {
		s.remote.stale()
	}
	var out []byte
	for i, snd := range rule.sends {
		r := s.remote
		if snd.files {
			r = s.files
		}
		cmd := []string{snd.word}
		if i < len(args) {
			cmd = append(cmd, args[i]...)
		}
		var err error
		switch snd.out {
		case outRead:
			out, err = r.Shell.Run(ctx, r.Script, script(cmd)...)
		case outPass:
			err = r.Shell.Pass(ctx, r.Script, script(cmd)...)
		case outFollow:
			err = r.Shell.Lines(ctx, r.Script, r.event, script(cmd)...)
		}
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// ports lists ws's wired ports.
func (s *Session) ports(ctx context.Context) ([]byte, error) { return s.do(ctx, opPorts) }

// waitPorts lists the ports once port holds 10.77.0.1, or after 10 s, in one login.
func (s *Session) waitPorts(ctx context.Context, port string) ([]byte, error) {
	return s.do(ctx, opWait, []string{port})
}

// nameCable names the port in ws's host_vars and runs the network and wireguard roles there.
func (s *Session) nameCable(ctx context.Context, port string) error {
	_, err := s.do(ctx, opNameCable, []string{port}, []string{"--only", "network,wireguard"})
	return err
}

// namePeer puts the Mac's key in ws's wg0 peer file and runs the wireguard role there.
func (s *Session) namePeer(ctx context.Context, key string) error {
	_, err := s.do(ctx, opNamePeer, []string{key}, []string{"--only", "wireguard"})
	return err
}

// nameUser names ws_user in ws's host_vars.
func (s *Session) nameUser(ctx context.Context, user string) error {
	_, err := s.do(ctx, opNameUser, []string{user})
	return err
}

// install runs install.sh of the version, from url, on ws.
func (s *Session) install(ctx context.Context, version, url string) error {
	_, err := s.do(ctx, opInstall, []string{version, url})
	return err
}

// run runs ws server apply with args.
func (s *Session) run(ctx context.Context, args []string) error {
	_, err := s.do(ctx, opRun, args)
	return err
}

// outbound names security_outbound: false in ws's host_vars.
func (s *Session) outbound(ctx context.Context) error {
	_, err := s.do(ctx, opOutbound)
	return err
}

// values reads ws's host_vars, as the files script prints it: empty where the file is missing.
func (s *Session) values(ctx context.Context) ([]byte, error) { return s.do(ctx, opValues) }

// setValues writes each KEY=VALUE pair of pairs into ws's host_vars in one login. A pair goes as
// base64, as the files script says.
func (s *Session) setValues(ctx context.Context, pairs []string) error {
	enc := make([]string, len(pairs))
	for i, p := range pairs {
		enc[i] = base64.StdEncoding.EncodeToString([]byte(p))
	}
	_, err := s.do(ctx, opSetValues, enc)
	return err
}
