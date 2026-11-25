# A Mac's servers in CI

The Mac's jobs work on one server named `ws`, at slot 0, as the Mac's side says
([The two files](../../macos/values/README.md)). The cases write its files through
[client-conf.sh](../client-conf.sh): `conf_write` splits the lines of stdin between `values.yml` and
`servers/NAME/values.yml`, and `conf_print` joins them. Both sit in `/etc/ws`, so `conf_write` writes
them through sudo. `conf_server` names the server, `ws` by
default, and a case that works on a second one sets it, then sets it back to `ws`.

A server's file that `conf_write` makes gets `WS_SLOT: 0` first, and `WS_TUNNEL_PORT: 51820` after it,
unless the lines hold them, as values.yml holds them. So a case that compares a file with the
lines it wrote lists no slot and no port.

Four cases of the `client` job work on a second server:

- `client/22-server.sh` makes a second server's file, `lab`, at slot 1, beside ws's. Then `ws macos
  values print` must stop, naming both servers. `--server box`, which has no file, and `--server
  Lab`, a bad name, must stop it too. `--server ws` must go on as with one server. The case's trap
  removes lab's directory.
- `client/90-clash.sh` makes the files `lab` and `box`, which clash with ws's. The bootstrap's plan
  must refuse each with its reason, and name `ws macos remove lab` beside it. It makes lab a
  guest of ws too. The plan must refuse a guest that holds a way of its own. It must refuse one
  that names no host, and one whose tunnel port is its host's.
- `client/92-second.sh` gives lab a third wireguard-go on UDP 51821. `mise -C /opt/ws/macos
  bootstrap` must then give lab a daemon and a utun of its own, at 10.99.1.1, and leave ws's daemon
  with its handshake. A second bootstrap must call no sudo.
- `client/94-remove.sh` runs `remove lab`: its files, daemon, host and `known_hosts` line must go, and
  ws's stay. The runner has no fnox key file, so remove must print its line that names lab's key,
  to drop it from the secrets by hand. `remove box` and `remove ws` must refuse, and the plan then
  exit 0.

Cases 90, 92 and 94 of the `client` job run near the end, before case 96's last bootstrap, as
`checks.sh` and the job stop at the first that fails.
