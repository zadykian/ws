# nvidia

The `nvidia` role installs NVIDIA's headless driver from Ubuntu's archive, on a machine with an
NVIDIA GPU. [ws-fans](fans.md) reads the GPU's temperature by its `nvidia-smi`. Its steps, in
`internal/server/nvidia`, are the GPU's check, the packages, and the driver's note of a reboot. It
runs before `fans`, and takes no values.

## The GPU

A PCI display device, of class `0x03`, from NVIDIA, vendor `0x10de`, makes the machine one with a
GPU. On Linux 7.0, nouveau reads no temperature from the RTX 5080's GB203, and NVIDIA drives it by
its open modules alone.

A machine without one, as a cloud VM, a VM on the Mac or CI's container, gets no driver. There,
`apt-get install -s` checks that the archive holds every name below, each flavor's modules
included, so that CI's run fails where one leaves it.

## The packages

The role pins the branch `595-server`, which `ubuntu-drivers install --gpgpu` picks for the GPU:

- `nvidia-headless-no-dkms-595-server-open`, the driver without X;
- `nvidia-utils-595-server`, which holds `nvidia-smi`;
- `linux-modules-nvidia-595-server-open-FLAVOR`, the modules, which Canonical builds and signs.

FLAVOR is that of the installed `linux-image-generic*`: `generic`, `generic-7.0`,
`generic-hwe-26.04` or `generic-hwe-26.04-edge`. Its metapackage pulls the modules of each new
kernel of the flavor. The role installs it at the version of the flavor's `linux-image-*`. A
newer one would pull a newer kernel without its headers, and DKMS would build no nct6687 for it.
A machine with none of the four fails the run. As Canonical signs the modules, Secure Boot needs
no key.

Unattended upgrades move the driver, as `resolute-security` carries it ([base](base.md)), and no
package is held. The role sets no persistence mode, as ws-fans' `nvidia-smi` holds the GPU open.
CUDA and the container toolkit, from NVIDIA's own repository, are not here.

## A reboot

The packages keep nouveau off and rebuild the initramfs, so their driver loads at the next boot.
While `/sys/module/nvidia/version` is not the version of the installed `nvidia-smi`, each run says
that a reboot is due. Until then, `nvidia-smi` fails on the mismatch, and ws-fans reads no GPU. An
upgrade of the driver leaves the same mismatch until the next reboot. Without a booted systemd, as
in CI's container, the note skips.
