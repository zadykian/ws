#!/bin/bash
# mise bootstrap (28-bootstrap.sh) unpacked the SDKs of macos/roles/dotnet's tree into
# /usr/local/share/dotnet, as root's. It wrote /etc/paths.d/dotnet, which puts that directory on a
# login shell's PATH. client-mise.sh made /usr/local/share root's, so sudo copied the tree in.

set -eo pipefail

fail=0
sdks=$(/usr/local/share/dotnet/dotnet --list-sdks | cut -d ' ' -f 1 | tr '\n' ' ')
[ "$sdks" = '10.0.404 11.0.100 ' ] ||
    {
        echo "::error::dotnet lists the SDKs '$sdks'; want 10.0.404 and 11.0.100 alone"
        fail=1
    }
[ "$(stat -f '%u' /usr/local/share/dotnet/dotnet)" = 0 ] ||
    {
        echo '::error::/usr/local/share/dotnet/dotnet is not root'"'"'s'
        fail=1
    }
[ "$(cat /etc/paths.d/dotnet)" = /usr/local/share/dotnet ] ||
    {
        echo '::error::/etc/paths.d/dotnet does not name /usr/local/share/dotnet'
        fail=1
    }
exit "$fail"
