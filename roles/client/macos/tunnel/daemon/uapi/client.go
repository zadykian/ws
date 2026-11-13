package uapi

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// SocketDir is where the device serves the UAPI socket of each tun, as wireguard-go does.
const SocketDir = "/var/run/wireguard"

// Socket returns the path of the UAPI socket of the tun called name.
func Socket(name string) string {
	return SocketDir + "/" + name + ".sock"
}

// Get asks the device behind the socket for its state and returns the text, as IpcGet does. It
// gives up at ctx's end, so that a daemon that is stuck does not hold status.
func Get(ctx context.Context, sock string) (string, error) {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", sock)
	if err != nil {
		return "", fmt.Errorf("cannot reach the device at %s: %w", sock, err)
	}
	defer func() { _ = conn.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	}
	if _, err := conn.Write([]byte("get=1\n\n")); err != nil {
		return "", fmt.Errorf("cannot ask the device: %w", err)
	}
	// The device ends its answer with a blank line, after errno=N.
	var text strings.Builder
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		if line == "errno=0" {
			continue
		}
		if strings.HasPrefix(line, "errno=") {
			return "", fmt.Errorf("the device answered %s", line)
		}
		text.WriteString(line + "\n")
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("cannot read the device's answer: %w", err)
	}
	return text.String(), nil
}
