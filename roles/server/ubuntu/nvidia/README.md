# nvidia

The `nvidia` role installs NVIDIA's headless driver from Ubuntu's archive, on a machine with an
NVIDIA GPU. [ws-fans](../fans/README.md) reads the GPU's temperature by its `nvidia-smi`. The role is
three units of [its spec](units.toml.tmpl), whose lines sit in `mise.server.toml`
([The frame](../frame/README.md)). It takes no values.

| Unit | What it does |
| --- | --- |
| `packages/nvidia-archive` | Checks that apt finds every name below, without the GPU, and installs none. |
| `packages/nvidia` | Installs the driver and each kernel flavor's modules, with the GPU. |
| `state/nvidia-reboot` | Says that a reboot is due while the kernel runs another driver, with the GPU. |

## The GPU

A PCI display device, of class `0x03`, from NVIDIA, vendor `0x10de`, makes the machine one with a
GPU: the units' `when = { gpu_vendor = "0x10de" }`
([when](../../../../engine/core/kind/README.md#the-specs)). On Linux 7.0, nouveau reads no
temperature from the RTX 5080's GB203, and NVIDIA drives it by its open modules alone.

A machine without one, as a cloud VM or a VM on the Mac, gets no driver. There the
two units say in a line that they are off, and read `in-sync`. The archive's check runs there
alone, by `when = { gpu_vendor = "!0x10de" }`: `apt-get install -s` of every name below, each
flavor's modules included, so that CI's run fails where one leaves the archive. It reads apt's lists
as mise's apt step left them. The GPU's machine turns it off, so a flavor that the machine does not
run, gone from the archive, never blocks the driver of the one it does.

## The packages

The role pins the branch `595-server`, which `ubuntu-drivers install --gpgpu` picks for the GPU:

- `nvidia-headless-no-dkms-595-server-open`, the driver without X;
- `nvidia-utils-595-server`, which holds `nvidia-smi`;
- `linux-modules-nvidia-595-server-open-FLAVOR`, the modules, which Canonical builds and signs.

The branch is named in three places: `packages.tmpl`, `archive.tmpl` and the reboot's `package` in
the spec. `TestNvidiaNamesOneBranch` fails where they differ.

FLAVOR is that of each installed `linux-image-generic*`: `generic`, `generic-7.0`,
`generic-hwe-26.04` or `generic-hwe-26.04-edge`. Its metapackage pulls the modules of each new
kernel of the flavor. The template installs it at the version of the flavor's `linux-image-*`,
read by `installed_version`. A newer one would pull a newer kernel without its headers, and DKMS
would build no nct6687 for it. A machine with none of the four fails the plan. As Canonical signs
the modules, Secure Boot needs no key.

Recommends follow apt's default, as mise's own apt does. Unattended upgrades move the driver, as
`resolute-security` carries it ([base](../base/README.md)), and no package is held. The role sets no
persistence mode, as ws-fans' `nvidia-smi` holds the GPU open. CUDA and the container toolkit,
from NVIDIA's own repository, are not here.

## A reboot

The packages keep nouveau off and rebuild the initramfs, so their driver loads at the next boot.
While `/sys/module/nvidia/version` is not the upstream version of the installed `nvidia-smi`, each
run warns that a reboot is due. Until then, `nvidia-smi` fails on the mismatch, and ws-fans reads
no GPU. An upgrade of the driver leaves the same mismatch until the next reboot. The note plans
nothing, so mise's plan stays in sync.

ws-fans waits on none of these units: it starts without the GPU, and reads it once the driver
loads.
