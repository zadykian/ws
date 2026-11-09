# tmux

The `tmux` role, Go steps in `internal/server/tmux`, installs tmux from Ubuntu's archive, with no
config, as cld runs tmux with `-f /dev/null`. Where tmux's snap is installed, the role removes it.
Where cld's sessions run the snap's tmux, end them before a real run. cld then finds
`/usr/bin/tmux` first on the `PATH`, while their key bindings call `/snap/bin/tmux`, which the
removal takes away.

Fresh asks for them by [a wrapper](fresh.md#the-wrapper). [lazygit's wrapper](lazygit.md#the-title)
sets a pane's title.

The [zed role](zed.md#terminals-in-tmux) reads a config of its own, with `-f`, on a socket of its
own, so the role's tmux still has none.

## ws-bar

`ws-bar.service` runs `ws server bar`, from a copy of ws in `/usr/local/lib/ws-bar`, as
[ws-fans](fans.md#ws-fans) does. It runs as a user that systemd makes as it starts, and every 10 s
it writes the host's load to `/run/ws-bar/line`, which tmux's status lines read:

```text
cpu 12% 52C  gpu 3% 40C  ram 41% 38C  nvme 44C  reboot
```

- **cpu:** the use since the line before, from `/proc/stat`, and the hwmon temperature of Intel's
  package (`coretemp`), or AMD's `Tctl` (`k10temp`).
- **gpu:** the highest use and temperature of the NVIDIA GPUs, which one child, `nvidia-smi -l 10`,
  streams with [the nvidia role's driver](nvidia.md). Before the driver loads, the bar tries it again
  each minute.
- **ram:** the memory in use, all but `MemAvailable`, and the hottest DIMM (`spd5118`).
- **nvme:** the hottest drive's `Composite` temperature.
- **reboot:** where `/run/reboot-required` says one is due, as after a new kernel.

A segment drops while its source gives nothing, so a VM shows the CPUs' use and the memory's. The
line goes as the service stops, so tmux shows no stale load. Without systemd running, as in CI's
container, the step skips. `ws server bar --interval 30s FILE` writes another file at another rate.
