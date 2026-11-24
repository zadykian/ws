# nvidia

The `nvidia` role gives a machine NVIDIA's headless driver from Ubuntu's archive, and gives a
container the card's userspace. A host holds the kernel modules and `nvidia-smi`, which
[ws-fans](../fans/README.md) reads for the GPU's temperature. A guest that is a container holds the
userspace alone, where its switch is on. A cloud VM with a GPU holds both, as a host does. The role
is units of [its spec](units.toml.tmpl), whose lines sit in `mise.host.toml` and `mise.guest.toml`
([The frame](../frame/README.md)). It takes one value, a guest's `WS_NVIDIA_GPU`.

| Unit | Where | What it does |
| --- | --- | --- |
| `packages/nvidia-archive` | Without the GPU | Checks that apt finds every name below, and installs none. |
| `packages/nvidia` | With the GPU, in no container | Installs the driver and each kernel flavor's modules. |
| `state/nvidia-reboot` | With the GPU, in no container | Says that a reboot is due while the kernel runs another driver. |
| `service/nvidia-devices` | The host, with the GPU | Waits for the device nodes before Incus starts a guest. |
| `packages/nvidia-userspace` | A guest's container, with the switch | Installs the userspace at the version of the host's module. |
| `file/nvidia-userspace-pin` | A guest's container, with the switch | Pins that version against an upgrade. |

## The GPU

A PCI display device, of class `0x03`, from NVIDIA, vendor `0x10de`, makes the machine one with a
GPU: the units' `when = { gpu_vendor = "0x10de" }`
([when](../../../engine/core/kind/README.md#the-specs)). On Linux 7.0, nouveau reads no
temperature from the RTX 5080's GB203, and NVIDIA drives it by its open modules alone.

A container shows its host's PCI devices, so the fact says nothing of the container's own kernel.
The driver's units add `container = "!yes"`, and a container takes its units by its switch instead
([A guest's card](#a-guests-card)).

A machine without a GPU, as a cloud VM without one, gets no driver. There the units say in a line
that they are off, and read `in-sync`. The archive's check runs there alone, by
`when = { gpu_vendor = "!0x10de" }`: `apt-get install -s` of every name below, each
flavor's modules included, so that CI's run fails where one leaves the archive. It reads apt's lists
as mise's apt step left them. The GPU's machine turns it off, so a flavor that the machine does not
run, gone from the archive, never blocks the driver of the one it does.

## The packages

The role pins the branch `595-server`, which `ubuntu-drivers install --gpgpu` picks for the GPU:

- `nvidia-headless-no-dkms-595-server-open`, the driver without X;
- `nvidia-utils-595-server`, which holds `nvidia-smi`;
- `linux-modules-nvidia-595-server-open-FLAVOR`, the modules, which Canonical builds and signs.

The branch is named in three places: `packages.tmpl`, `archive.tmpl` and the reboot's `package` in
the spec. `TestNvidiaNamesOneBranch` fails where they differ. `userspace.tmpl` and `pin.tmpl` name
`nvidia-utils-595-server` too, and the container tests' `nvidiaSMI` catches a drift there.

FLAVOR is that of each installed `linux-image-generic*`: `generic`, `generic-7.0`,
`generic-hwe-26.04` or `generic-hwe-26.04-edge`. Its metapackage pulls the modules of each new
kernel of the flavor. The template installs it at the version of the flavor's `linux-image-*`,
read by `installed_version`. A newer one would pull a newer kernel without its headers, and DKMS
would build no nct6687 for it. A machine with none of the four fails the plan. As Canonical signs
the modules, Secure Boot needs no key.

Recommends follow apt's default, as mise's own apt does. Unattended upgrades move the driver, as
`resolute-security` carries it ([base](../base/README.md)), and no package is held on a machine with
a kernel of its own. The role sets no persistence mode, as ws-fans' `nvidia-smi` holds the GPU open.
CUDA and the container toolkit, from NVIDIA's own repository, are not here.

## A reboot

The packages keep nouveau off and rebuild the initramfs, so their driver loads at the next boot.
While `/sys/module/nvidia/version` is not the upstream version of the installed `nvidia-smi`, each
run warns that a reboot is due. Until then, `nvidia-smi` fails on the mismatch, and ws-fans reads
no GPU. An upgrade of the driver leaves the same mismatch until the next reboot. The note plans
nothing, so mise's plan stays in sync. A container has no such note: its kernel is the host's.

ws-fans waits on none of these units: it starts without the GPU, and reads it once the driver
loads.

## A guest's card

A container shares its host's kernel and loads no module, so the host holds the modules and the
guest the userspace. The guest's own values hold the switch, in
`/etc/ws/guests/NAME/values.yml` on the host ([incus](../incus/README.md#the-guests)):

```yaml
WS_NVIDIA_GPU: true
```

A host's file takes no such key. Where the switch is on, the host gives the guest a `gpu` device by
`vendorid=10de` ([A guest's instance](../incus/README.md#a-guests-instance)). Several guests may
hold the card at once, time-sliced and with no memory isolation between them. With the switch off, the
device is removed. The host's profile does not set `nvidia.runtime`, as it needs NVIDIA's container
toolkit on the host.

In the container, `packages/nvidia-userspace` installs the branch's userspace package at the
upstream version that the host's `/sys/module/nvidia/version` gives. `file/nvidia-userspace-pin`
writes `/etc/apt/preferences.d/ws-nvidia`, which pins that version at priority 1001. An upgrade of
the host's driver moves the host's `nvidia-smi` and its module only at the host's reboot. Until
then the guest keeps the old userspace, which matches the module that runs. A run of the guest's
setup after the reboot pins the new version, in the units' order. Where the host loads no module,
the plan fails and names the switch: the card cannot reach the guest. The container has no
`state/nvidia-reboot` or driver.

## The device nodes

Incus passes `/dev/nvidia*` to a guest only as they exist when the guest starts. The host's
`ws-nvidia-devices.service` is a oneshot that waits up to a minute for `/dev/nvidia0` and
`/dev/nvidia-uvm`. The drop-in `incus.service.d/ws-nvidia.conf` wants it and orders `incus.service`
after it. The unit never fails, so no missing card keeps Incus down. A host whose module is not
loaded, as before the reboot, waits for nothing. A missing node gets a line in the journal. The unit
runs where the host holds the GPU alone.
