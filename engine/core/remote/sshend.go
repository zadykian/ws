package remote

import (
	"context"
	"io/fs"
)

// End ends the masters that Run opened in the lock, before the lock goes. It reports nothing. Each
// call names its socket's file, direct or a server's name, as ssh reads no config here and takes
// the word only as the call's host.
func (s SSH) End(ctx context.Context) {
	names, err := s.M.ReadDir(s.Lock)
	if err != nil {
		return
	}
	for _, n := range names {
		sock := s.Lock + "/" + n
		if info, err := s.M.Stat(sock); err == nil && info.Mode&fs.ModeSocket != 0 {
			_, _ = s.M.Run(ctx, "ssh", "-F", "/dev/null", "-O", "exit", "-o", "ControlPath="+sock, n)
		}
	}
}
