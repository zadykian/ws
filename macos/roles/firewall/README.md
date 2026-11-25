# The Mac's firewall

`mise bootstrap` turns macOS's Application Firewall on and has it block all incoming connections,
by the unit `state/firewall` of the role `firewall`:

```sh
/usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on --setblockall on
```

## Why

A cloud VM ends its own tunnel and holds no filter of its own
([No filter](../../../ubuntu/roles/security/README.md#no-filter)), so root there can open a
connection to the Mac through the tunnel. The Mac's own firewall keeps it from.

## The unit

- The command runs as root, through sudo. The check runs as the user, with no sudo, so a run that
  changes nothing asks no password.
- socketfilterfw's get options exit 0 whatever the state, so the check reads their text. It exits
  other than 0 unless `--getglobalstate` ends in `State = 1` or `State = 2` and `--getblockall`
  says `blocking all`, and the command runs then. On macOS 26, block-all prints `State = 2`, and
  a firewall that only filters `State = 1`.
- To see both states, run `/usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate`
  and then the same with `--getblockall`.

## What keeps working

The Mac speaks first on the tunnel and the cable, so block-all does not stop them: a reply to an
outgoing packet passes. The firewall spares loopback too, which CI's cases lean on after
[`28-bootstrap.sh`](../../../.github/client/28-bootstrap.sh) turns it on.

## Turning it off

A firewall turned off by hand comes back at the next `mise bootstrap`. Block-all ignores the
allow list of programs. While the unit holds, the Mac's sharing services, Screen Sharing, File
Sharing and AirPlay among them, get no connection in. Only the basic services, such as discovery
and IPsec, still do.
