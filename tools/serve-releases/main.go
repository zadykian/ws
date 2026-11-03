// Command serve-releases serves a release layout as GitHub does, for the cases of CI's jobs that
// install ws.
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// handler serves dir's files, and the redirect of /latest.
func handler(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest" {
			files.ServeHTTP(w, r)
			return
		}
		tag, err := os.ReadFile(filepath.Join(dir, "latest")) //nolint:gosec // dir is the command's own argument
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "http://"+r.Host+"/tag/"+strings.TrimSpace(string(tag)), http.StatusFound) //nolint:gosec // the stand-in answers the host that asked, as GitHub does
	})
}

// newCommand returns serve-releases, which serves until it fails.
func newCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve-releases DIR PORT [PORT_FILE]",
		Short: "serve a release layout as GitHub does",
		Long: "DIR holds download/TAG/FILE and latest, a file with the tag that /latest answers with: the\n" +
			"redirect GitHub gives, to /tag/TAG. A case changes the latest release by writing latest. PORT\n" +
			"is the port on loopback, or 0 for the one the system picks. The port goes to PORT_FILE, if\n" +
			"given, once the server listens.",
		Args:          cobra.RangeArgs(2, 3),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(_ *cobra.Command, args []string) error {
			portFile := ""
			if len(args) == 3 {
				portFile = args[2]
			}
			return serve(args[0], args[1], portFile)
		},
	}
}

// serve listens on loopback's port, writes the port it got to portFile where one is given, and
// serves dir.
func serve(dir, port, portFile string) error {
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		return err
	}
	if portFile != "" {
		got := ln.Addr().(*net.TCPAddr).Port
		if err := os.WriteFile(portFile, fmt.Appendf(nil, "%d\n", got), 0o600); err != nil {
			_ = ln.Close()
			return err
		}
	}
	srv := &http.Server{Handler: handler(dir), ReadHeaderTimeout: time.Minute}
	return srv.Serve(ln)
}

func main() {
	if err := newCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "serve-releases:", err)
		os.Exit(1)
	}
}
