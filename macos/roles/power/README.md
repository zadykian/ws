# The Mac's power

`mise bootstrap` keeps the display on while the Mac is on power, by a unit of the role `power`. It
sets one fact, as a `state` unit that runs a command
([The state kind](../../../engine/core/kind/state.md)). The check runs as the user, and the
command as root through sudo, so a run that changes nothing asks no password.

## state/power-display

```sh
pmset -c displaysleep 0
```

`-c` is the mains' setting alone, so the battery keeps its own sleep. The check reads
`pmset -g custom`, which lists each source's settings under its heading, and wants `displaysleep 0`
under `AC Power:`. The command runs where that line is missing or reads another number. A Mac on
power then never turns its display off, as a session that the server's work keeps open needs.
