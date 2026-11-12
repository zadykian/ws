package wizard

import "context"

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
