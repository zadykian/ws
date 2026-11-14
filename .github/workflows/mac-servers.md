# A Mac's servers in CI

The Mac's jobs work on one server named `ws`, at slot 0, as the Mac's side says
([The two files](../../roles/client/macos/conf/README.md)). The cases write its files through
[mac-conf.sh](../mac-conf.sh): `conf_write` splits the lines of stdin between `mac.conf` and
`servers/NAME/server.conf`, and `conf_print` joins them. `conf_server` names the server, `ws` by
default, and a case that works on a second one sets it, then sets it back to `ws`.

A server's file that `conf_write` makes gets `WS_SLOT=0` first, and `WS_PORT=51820` after it, unless
the lines hold them, as the wizard writes them. So a case that compares a file with the wizard's
lines lists no slot and no port.

Five cases of the `mac` job and two of the wizard's work on a second server:

- `mac/22-server.sh` makes a second server's file, `lab`, at slot 1, beside ws's. Then `ws client macos
  apply --check` must stop, naming both servers. `--server box`, which has no file, and `--server
  Lab`, a bad name, must stop it too. `--server ws` must go on as with one server. The case's trap
  removes lab's directory.
- `mac/90-clash.sh` makes the files `lab` and `box`, which clash with ws's. `apply --check
  --server lab` must refuse each with its reason. `--server ws` must name `ws client macos remove`
  beside a refused file.
- `mac/92-second.sh` gives lab a third wireguard-go on UDP 51821. `apply --server lab` must then
  give lab a daemon and a utun of its own, at 10.99.1.1, and leave ws's daemon with its handshake.
- `mac/94-remove.sh` runs `remove lab`: its files, daemon, host and `known_hosts` line must go, and
  ws's stay. `remove box` and `remove ws` must refuse.
- `mac-wizard/16-server.sh` runs the wizard with a bad name, which must stop it.
  On a `mac.conf` with no server, a run with no terminal and no name must stop and ask for
  `--server`. There `--check` with a name must show that server's file headed by `WS_SLOT=0`, and
  write none.
- `mac-wizard/62-second.sh` runs the wizard for lab, the last case. `standin.sh` makes a second
  stand-in server for it at `127.0.0.1:22023`, with files and logs of its own, so that ws's counts
  of runs hold. Its wg0 listens on the `wireguard_port` of its host_vars. On lab's file made by
  hand, which holds `WS_USER` alone, `--check` must show `WS_SLOT=1` and leave the file. The run
  must give lab slot 1, port 51821, a host_vars of 10.99.1.1/24 and 51821, a daemon with a
  handshake, and one login through the tunnel. It must leave ws's files as they were. The next run
  must print nothing.

The new cases run last, as `checks.sh` and the job stop at the first that fails.
