"""Serves a release layout as GitHub does, for the cases of CI's jobs that install ws.

    python3 release-standin.py DIR PORT [PORT_FILE]

DIR holds download/TAG/FILE and latest, a file with the tag that /latest answers with: the redirect
GitHub gives, to /tag/TAG. A case changes the latest release by writing latest. PORT is the port on
loopback, or 0 for the one the system picks. The port goes to PORT_FILE, if given, once the server
listens.
"""

import functools
import http.server
import pathlib
import sys


class Releases(http.server.SimpleHTTPRequestHandler):
    """Serves DIR's files, and the redirect of /latest."""

    def do_GET(self):
        """Answers /latest with the redirect, and any other path with its file."""
        if self.path == "/latest":
            tag = (pathlib.Path(self.directory) / "latest").read_text(encoding="utf-8").strip()
            self.send_response(302)
            self.send_header("Location", f"http://{self.headers['Host']}/tag/{tag}")
            self.end_headers()
            return
        super().do_GET()


def main():
    """Serve until killed."""
    directory, port, *port_file = sys.argv[1:]
    handler = functools.partial(Releases, directory=directory)
    with http.server.ThreadingHTTPServer(("127.0.0.1", int(port)), handler) as server:
        for file in port_file:
            pathlib.Path(file).write_text(f"{server.server_address[1]}\n", encoding="utf-8")
        server.serve_forever()


main()
