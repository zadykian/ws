# fans

The `fans` role drives the server's fans on curves of its own. It builds the board's fan driver
with DKMS, and runs `ws-fans`, a service of its own, on the curves of its variables. The role is
three units of [its spec](units.toml.tmpl), whose lines sit in `mise.server.toml`
([The frame](../frame/README.md)). The daemon is Go, `ws server fans`, in `daemon/`.

| Unit | What it does |
| --- | --- |
| `packages/fans` | Installs `dkms`, `linux-headers-generic` and the running kernel's headers. |
| `dkms/fans` | Builds the driver `nct6687` with DKMS, after the packages, and loads it. |
| `service/fans` | Writes the four files below, and keeps `ws-fans` enabled and running. |

Each runs where DMI names the board's maker `WS_FANS_BOARD_VENDOR`, MSI: the units'
`when = { board_vendor = ... }` ([when](../../../../engine/core/kind/README.md#the-specs)). A
machine with another board is left alone, as a cloud VM or a VM on the Mac. There
each unit says in a line that it's off, and reads `in-sync`. With the value empty, the line says
that the `when` gives no maker.

## The driver

The board is MSI's, with a Nuvoton NCT6687D for its fan headers. Linux's own `nct6683` reads that
chip but sets none of its fans on MSI's boards. So the role builds `nct6687`, from
[Fred78290/nct6687d](https://github.com/Fred78290/nct6687d), with DKMS: the unit `dkms/fans`
([the kind dkms](../../../../engine/core/kind/README.md#dkms)). That repository tags no releases:
the spec pins a commit, and the SHA-256 of GitHub's tarball of it.

The unit downloads the tarball against that SHA-256, and unpacks it beside
`/usr/src/nct6687d-COMMIT`. It moves the tree in whole, so that a run cut short leaves no half
tree. The tree's `.ws-source` file holds the tarball's URL. DKMS's version is the commit's first 12
characters, which `files/dkms.conf` gives in place of the driver's fixed `1`, and the unit builds
it for the running kernel. DKMS builds it again for each new kernel, from the headers
`linux-headers-generic` installs beside it. A new pin removes the old version from DKMS, and its
source. `dkms status` says which versions DKMS holds. `/etc/modprobe.d/ws-fans.conf` keeps
`nct6683` off the chip, and `/etc/modules-load.d/ws-fans.conf` loads `nct6687` at boot. The run
unloads a loaded `nct6683`, as after `sensors-detect`, and loads `nct6687`.

### Secure Boot

With Secure Boot on, the kernel loads a module only with a key it trusts. DKMS signs `nct6687`
with Ubuntu's machine owner key, `/var/lib/shim-signed/mok/MOK.der`, which the firmware trusts once
enrolled. Until then, the run says the kernel refused the driver and goes on, and the board's own
curves drive its fans. `/run/ws-refused-nct6687` marks the refusal until the reboot, so the runs
between plan no load. To enroll the key, once, at the server's console:

```sh
mokutil --sb-state                                      # SecureBoot enabled
mokutil --import /var/lib/shim-signed/mok/MOK.der       # asks for a one-time password
reboot                                                  # MOK manager: Enroll MOK, the password
```

Then run the bootstrap again, as
[the runbook](../../runbook.md) writes it, or `modprobe nct6687 && systemctl restart ws-fans`.

## ws-fans

`ws-fans.service` runs `ws server fans` from `/usr/local/bin/ws`. It reads `/etc/ws/fans.json`,
which the unit writes from its variables, below. The daemon's own loader checks a new config as the
unit plans, so a config it refuses fails the plan and replaces nothing. A change to the config, the
unit or the modules' files restarts the service. So does a new ws: the running process's binary is
then the replaced one, which the unit reads as drift. The log, `journalctl -u ws-fans`, says when a
sensor or a fan goes missing or comes back.

Every second it reads each sensor, moves each curve, and steps each fan towards its curve's speed:

- **Sensors:** a hwmon channel by its label, or an NVIDIA GPU's temperature, which one child,
  `nvidia-smi -l 1`, streams to the daemon. Or the average of another sensor over a time, or the
  highest of several.
- **Curves:** a graph of points, `[°C, %]`, flat before the first and past the last. Its speed
  follows the temperature once it has moved the hysteresis, in °C, from where the speed last
  changed, and stayed there for the response time, in seconds. Up and down each have their own. At
  or beyond the first or the last point, the speed follows at once.
- **Fans:** a header of the board, `pwmN` of the channel whose `fanN_label` is named. Each runs on
  a curve or at a fixed speed, and moves 10% a second at most.

A fan whose temperature or device is missing goes back to the board's own curve until both return.
So does every fan when the service stops. As it starts, it takes each fan from the speed it runs at.

The RTX 5080's own fans are off the card, which the GPU Fans on the board's System Fan #1 cool. So
ws-fans sets no fan through NVIDIA's driver: it reads only the GPU's temperature, by `nvidia-smi`,
which keeps ws free of NVML's library. That needs NVIDIA's driver, which [nvidia](../nvidia/README.md)
installs. Without it, the GPU Fans and the Exhaust Fans, on the GPU's temperature, keep the board's
curves.

## The defaults

`roles/server/values/default_fans.go` sets these sensors:

- `cpu`: coretemp's `Package id 0`, the CPU package's temperature;
- `gpu`: GPU 0's temperature;
- `cpu_average` and `gpu_average`: each of the two, averaged over 60 s;
- `cpu_gpu_max`: the higher of the two averages.

Each curve has a hysteresis of 2 °C and a response time of 5 s, up and down:

| Curve         | Sensor        | Points, °C: %          |
| ------------- | ------------- | ---------------------- |
| `cpu`         | `cpu_average` | 50: 30, 80: 60         |
| `gpu`         | `gpu_average` | 30: 20, 50: 20, 80: 60 |
| `cpu_gpu_max` | `cpu_gpu_max` | 50: 30, 80: 60         |

The fans are the board's headers, by `nct6687`'s labels:

| Fan          | Header        | Speed               |
| ------------ | ------------- | ------------------- |
| CPU Pump     | Pump Fan      | on `cpu`            |
| Exhaust Fans | CPU Fan       | on `cpu_gpu_max`    |
| GPU Fans     | System Fan #1 | on `gpu`            |
| MB Fan       | System Fan #2 | 0%                  |

System Fan #3 to #6 are left out, so the board's curves drive them.

## Variables

- `WS_FANS_BOARD_VENDOR`: the board's maker, as DMI names it, where the role drives the fans.
  Empty, it drives none.
- `WS_FANS_SENSORS`, `WS_FANS_CURVES` and `WS_FANS_CONTROLS`: the sensors, the curves and the fans,
  as above. Each sensor names only sensors above it.
- `WS_FANS_STEP`: how far a fan's speed moves at most in a tick, up and down, in %.
- `WS_FANS_INTERVAL`: the tick, in seconds.

With `nct6687` loaded, node_exporter reads the fans' speeds too, for SigNoz ([otelcol](../otelcol/README.md)).
`mise bootstrap plan` builds no driver, and says whether a unit would change.
