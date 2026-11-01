"""Serves a release layout as GitHub does, for the install job's cases.

    python3 install-server.py DIR TAG PORT_FILE

DIR holds download/TAG/FILE. /latest answers with the redirect GitHub gives, to /tag/TAG of the
TAG named. The port, which the system picks, goes to PORT_FILE once the server listens.
"""

import functools
import http.server
import pathlib
import sys


class Releases(http.server.SimpleHTTPRequestHandler):
    """Serves DIR's files, and the redirect of /latest."""

    latest = ""

    def do_GET(self):
        """Answers /latest with the redirect, and any other path with its file."""
        if self.path == "/latest":
            self.send_response(302)
            self.send_header("Location", f"http://{self.headers['Host']}/tag/{self.latest}")
            self.end_headers()
            return
        super().do_GET()


def main():
    """Serve until killed."""
    directory, Releases.latest, port_file = sys.argv[1:4]
    handler = functools.partial(Releases, directory=directory)
    with http.server.ThreadingHTTPServer(("127.0.0.1", 0), handler) as server:
        pathlib.Path(port_file).write_text(f"{server.server_address[1]}\n", encoding="utf-8")
        server.serve_forever()


main()
