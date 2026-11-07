# The server's wizard

`sudo ws server wizard` runs the stages of [the Mac's wizard](mac-wizard.md) that act on ws, on ws
itself, for a login over plain ssh, such as Termius on an iPad. It runs as root alone, and `--check`
prints what each stage would do and changes nothing. At a terminal the stages run on
[the run screen](tui.md#the-run-screen), and `--plain` or no terminal keeps the lines. It holds the
lock `ws-wizard.lock` in `$TMPDIR`, as the Mac's wizard does, and a run in which a stage acted ends
with what stays by hand of ws.

1. ws's values: each key that `/etc/ws/host_vars.yml` lacks. The target comes first: `lan` for a
   machine on the LAN or a VM on the Mac, which keeps the outbound table on, and `cloud` for a cloud
   VM, which names `security_outbound: false`. Then `ws_user`, by default the user that ran sudo.
   Root takes it on a cloud VM alone, which may have no other login. Each answer is written, a
   default too, by `ws server values set`'s code, so that a later run from either side asks nothing.
   With no terminal, the stage stops and names `sudo ws server values set KEY VALUE`.
2. ws's run: `ws server apply` in a unit of its own, by the remote script that the Mac's wizard
   sends over ssh, here on `sh`'s standard input. The unit goes on where the login drops, and reads
   [ws's secrets file](mac-secrets.md#wss-file) where its credential stands beside it. A run follows
   the unit that runs, whichever wizard started it.

ws's run is done where the last full run exited 0, by this ws, with the secrets file unchanged
since. Both wizards read that one stamp, so a run from either side after the other's, with nothing
changed, applies nothing. ws's values name its target, which the stage reads in place of
`mac.conf`. With no booted systemd, as in a container, the stage says so and skips: run
`sudo ws server apply` there.

The Mac's other stages stay the Mac's: the tunnel, the cable, claude's login and the signing key
([By hand](../README.md#by-hand)). The list at the end names ws's fans' driver where Secure Boot
refused it.
