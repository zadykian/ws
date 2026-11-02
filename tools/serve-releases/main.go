// Command serve-releases serves a release layout as GitHub does, for the cases of CI's jobs that
// install ws:
//
//	serve-releases DIR PORT [PORT_FILE]
//
// DIR holds download/TAG/FILE and latest, a file with the tag that /latest answers with: the
// redirect GitHub gives, to /tag/TAG. A case changes the latest release by writing latest. PORT is
// the port on loopback, or 0 for the one the system picks. The port goes to PORT_FILE, if given,
// once the server listens.
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func main() {
	if n := len(os.Args); n < 3 || n > 4 {
		fmt.Fprintln(os.Stderr, "usage: serve-releases DIR PORT [PORT_FILE]")
		os.Exit(2)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:"+os.Args[2])
	if err == nil && len(os.Args) == 4 {
		port := ln.Addr().(*net.TCPAddr).Port
		err = os.WriteFile(os.Args[3], fmt.Appendf(nil, "%d\n", port), 0o600) //nolint:gosec // the command's own argument
	}
	if err == nil {
		srv := &http.Server{Handler: handler(os.Args[1]), ReadHeaderTimeout: time.Minute}
		err = srv.Serve(ln)
	}
	fmt.Fprintln(os.Stderr, "serve-releases:", err)
	os.Exit(1)
}
