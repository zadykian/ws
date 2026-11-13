# A Mac's servers in CI

The Mac's jobs work on one server named `ws`, at slot 0, as the Mac's side says
([The two files](../../roles/client/macos/conf/README.md)). The cases write its files through
[mac-conf.sh](../mac-conf.sh): `conf_write` splits the lines of stdin between `mac.conf` and
`servers/NAME/server.conf`, and `conf_print` joins them. `conf_server` names the server, `ws` by
default, and a case that works on a second one sets it, then sets it back to `ws`.

A server's file that `conf_write` makes gets `WS_SLOT=0` first, unless the lines hold it, as the
wizard writes it. So a case that compares a file with the wizard's lines lists no slot.

Two cases check `--server`, one a job:

- `mac/22-server.sh` makes a second server's file, `lab`, at slot 1, beside ws's. Then `ws client macos
  apply --check` must stop, naming both servers. `--server box`, which has no file, and `--server
  Lab`, a bad name, must stop it too. `--server ws` must go on as with one server. The case's trap
  removes lab's directory.
- `mac-wizard/16-server.sh` runs the wizard with a bad name, and with `lab` beside ws: both must
  stop it. On a `mac.conf` with no server, a run with no terminal and no name must stop and ask for
  `--server`. There `--check` with a name must show that server's file headed by `WS_SLOT=0`, and
  write none.
