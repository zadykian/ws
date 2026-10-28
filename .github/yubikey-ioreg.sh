#!/bin/sh
# A stand-in ioreg for the mac and wizard jobs, which yubikey-standin.sh's setup links as ioreg in
# $RUNNER_TEMP/yubikey/bin, first on the jobs' PATH. It answers ws-agent's call alone,
# ioreg -p IOUSB -l -w 0, as macOS prints the USB plane: a keyboard, and a YubiKey 5C Nano for each
# of the stand-in's YubiKeys plugged in. A Nano's registry ID is its YubiKey's id, which the
# stand-in changes at each rebuild of its token, as a replug changes a real Nano's. So an agent that
# runs loads the keys again after a key is made or dropped, as after a replug.

set -eu
S=${RUNNER_TEMP:?the stand-in YubiKey lives in RUNNER_TEMP}/yubikey

[ "$*" = '-p IOUSB -l -w 0' ] || {
    echo "ioreg: the stand-in takes no $*" >&2
    exit 2
}
printf '%s\n' '+-o Root  <class IORegistryEntry, id 0x100000100, retain 31>' \
    '  +-o AppleT8132USBXHCI@01000000  <class AppleT8132USBXHCI, id 0x100000356, registered>' \
    '    +-o Keyboard@01100000  <class IOUSBHostDevice, id 0x100000a07, registered, retain 22>' \
    '      {' '        "USB Product Name" = "Keyboard"' '        "idVendor" = 1452' '      }'
for each in "$S"/*/in; do
    [ -f "$each" ] || continue
    nano="    +-o YubiKey OTP+FIDO+CCID@01200000  <class IOUSBHostDevice, id $(cat "${each%in}id")"
    printf '%s\n' "$nano, registered, retain 26>" '      {' \
        '        "USB Product Name" = "YubiKey OTP+FIDO+CCID"' \
        '        "USB Vendor Name" = "Yubico"' '        "idVendor" = 4176' '      }'
done
