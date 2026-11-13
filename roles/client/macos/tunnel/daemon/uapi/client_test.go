package uapi

import (
	"bufio"
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// serve answers one connection on a socket in a new directory with reply, and returns the socket's
// path and the request it got.
func serve(t *testing.T, reply string) (string, <-chan string) {
	dir, err := os.MkdirTemp("", "uapi")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sock := filepath.Join(dir, "s.sock")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	got := make(chan string, 1)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		r := bufio.NewReader(conn)
		first, _ := r.ReadString('\n')
		_, _ = r.ReadString('\n')
		got <- first
		_, _ = conn.Write([]byte(reply))
	}()
	return sock, got
}

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		reply   string
		want    string
		wantErr string
	}{
		{"peer", "private_key=aa\nrx_bytes=5\nerrno=0\n\n", "private_key=aa\nrx_bytes=5\n", ""},
		{"empty", "errno=0\n\n", "", ""},
		{"errno", "errno=22\n\n", "", "errno=22"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sock, got := serve(t, tt.reply)
			text, err := Get(context.Background(), sock)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want one with %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if text != tt.want {
				t.Errorf("text = %q, want %q", text, tt.want)
			}
			if req := <-got; req != "get=1\n" {
				t.Errorf("request = %q, want get=1", req)
			}
		})
	}
}

func TestGetMissing(t *testing.T) {
	_, err := Get(context.Background(), filepath.Join(t.TempDir(), "none.sock"))
	if err == nil || !strings.Contains(err.Error(), "cannot reach the device") {
		t.Errorf("err = %v, want cannot reach the device", err)
	}
}

func TestSocket(t *testing.T) {
	if got := Socket("utun4"); got != "/var/run/wireguard/utun4.sock" {
		t.Errorf("Socket = %q", got)
	}
}
