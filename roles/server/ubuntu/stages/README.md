# The server's wizard

`sudo ws server ubuntu wizard` runs the stages of [the Mac's wizard](../../../client/macos/stages/README.md)
that act on ws, on ws itself, for a login over plain ssh, such as Termius on an iPad. It runs as
root alone, and `--check` prints what each stage would do and changes nothing. At a terminal it
opens [the wizard's list](../../../../engine/tui/internal/queue/README.md), one, `ws`, as every stage runs on ws. `s`
opens ws's state, `e` the forms of `sudo ws server ubuntu configure` ([The Configure
screen](../../../../engine/tui/internal/configure/README.md#on-ws)), and the stages
chosen run. `--check` goes straight to the run screen, and `--plain` or no terminal keeps the lines.
It holds the lock `ws-wizard.lock` in `$TMPDIR`, as the Mac's wizard does, and a run in which a
stage acted ends with what stays by hand of ws.

1. ws's values: each key that `/etc/ws/host_vars.yml` lacks. The target comes first: `lan` for a
   machine on the LAN or a VM on the Mac, which keeps the outbound table on, and `cloud` for a cloud
   VM, which names `security_outbound: false`. Then `ws_user`, by default the user that ran sudo.
   Root takes it on a cloud VM alone, which may have no other login. Each answer is written, a
   default too, by `ws server ubuntu values set`'s code, so that a later run from either side asks
   nothing. With no terminal, the stage stops and names `sudo ws server ubuntu values set KEY
   VALUE`.
2. ws's run: `ws server ubuntu apply` in a unit of its own, by the remote script that the Mac's
   wizard sends over ssh, here on `sh`'s standard input. The unit goes on where the login drops, and
   reads [ws's secrets file](../secrets/README.md#wss-file) where its credential stands beside it. A run
   follows the unit that runs, whichever wizard started it.
3. claude on ws: `WS_USER`'s claude, logged in, as `claude auth status` says. Else the stage runs
   `runuser -l WS_USER -c 'claude auth login'` at the terminal: open the address it shows in a
   browser. With no terminal, it stops. Where ws lacks claude, it names `sudo ws server ubuntu
   apply`.
4. Signing key: done where `WS_USER`'s git config names no signing key file. Else GitHub must list
   ws's public signing key for `WS_USER`'s GitHub login: its gh's, else its git config's
   `github.user`. GitHub's API lists a login's signing keys with no token. ws's gh has no account
   permission ([By hand](../../../../README.md#by-hand)), so the list at the end names the key, to add on
   GitHub as a signing key.

ws's run is done where the last full run exited 0, by this ws, with the secrets file and
`host_vars.yml` unchanged since. Both wizards read that one stamp, so a run from either side after
the other's, with nothing changed, applies nothing. A change to `host_vars.yml`, by the values
stage, `ws server ubuntu values set`, Configure or by hand, makes the next run due. ws's values name
its target, which the stage reads in place of `mac.conf`. With no booted systemd, as in a container,
the stage says so and skips: run `sudo ws server ubuntu apply` there. The remote script reads the
claude and the signing key of `WS_USER`, host_vars' `ws_user`, where the Mac's reads those of the
user that ssh logs in as.

The Mac's other stages stay the Mac's: the tunnel and the cable. The list at the end names ws's
fans' driver where Secure Boot refused it, and the router's static lease on the LAN. It names the
signing key where GitHub lacks it. The wizard adds no device's key to ws, so the list names
`ssh-import-id gh:LOGIN`, run as `WS_USER`, for each one
([SSH and the firewall](../security/README.md#ssh-and-the-firewall)).
