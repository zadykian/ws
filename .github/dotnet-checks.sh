#!/bin/bash
# The server job's check of devtools' .NET, run as root after setup. dotnet is ws's link, before
# the image's own on PATH, and lists the SDKs of devtools' tree alone. roslyn-language-server's own
# program finds that .NET by /etc/dotnet/install_location, with no DOTNET_ROOT. With no flag, it
# asks for --stdio.
set -euo pipefail

sdks=$(dotnet --list-sdks | cut -d ' ' -f 1 | tr '\n' ' ')
echo "SDKs: $sdks"
[ "$sdks" = "10.0.404 11.0.100 " ]
out=$(timeout 60 env -u DOTNET_ROOT /root/.dotnet/tools/roslyn-language-server 2>&1 || true)
head -n 3 <<<"$out"
case $out in *--stdio*) ;; *) exit 1 ;; esac
