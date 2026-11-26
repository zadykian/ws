#!/bin/bash
# mise bootstrap (28-bootstrap.sh) wrote macos/roles/desktop's preferences, which later plans call
# in sync. This case reads some back with their types, as defaults read-type prints them, the
# current host's screen saver delay among them. The Dock and Finder run on: mise restarts neither.

set -eo pipefail

fail=0
# check DOMAIN KEY TYPE VALUE, the type as defaults read-type prints it.
check() {
    local type value
    type=$(defaults read-type "$1" "$2" 2>/dev/null | sed 's/^Type is //') || type=
    value=$(defaults read "$1" "$2" 2>/dev/null) || value=
    if [ "$type" != "$3" ] || ! awk -v a="$value" -v b="$4" -v t="$3" \
        'BEGIN { exit !((t == "float" || t == "integer") ? a + 0 == b + 0 : a == b) }'; then
        echo "::error::$1 $2 is $value ($type), want $4 ($3)"
        fail=1
    fi
}

while IFS='|' read -r domain key type value; do
    check "$domain" "$key" "$type" "$value"
done <<'EOT'
com.apple.dock|tilesize|integer|74
com.apple.dock|show-recents|boolean|0
com.apple.dock|minimize-to-application|boolean|1
com.apple.finder|FXPreferredViewStyle|string|clmv
com.apple.finder|NewWindowTarget|string|PfHm
com.apple.WindowManager|EnableTilingByEdgeDrag|boolean|0
com.apple.WindowManager|StandardHideWidgets|boolean|1
com.apple.screencapture|target|string|clipboard
NSGlobalDomain|AppleInterfaceStyle|string|Dark
NSGlobalDomain|AppleInterfaceStyleSwitchesAutomatically|boolean|0
NSGlobalDomain|NSGlassTintAmount|float|1
NSGlobalDomain|com.apple.sound.beep.volume|float|0
NSGlobalDomain|com.apple.trackpad.scaling|float|0.5
NSGlobalDomain|com.apple.sound.uiaudio.enabled|integer|0
EOT

# The screen saver's delay is the current host's.
type=$(defaults -currentHost read-type com.apple.screensaver idleTime 2>/dev/null | sed 's/^Type is //') || type=
idle=$(defaults -currentHost read com.apple.screensaver idleTime 2>/dev/null) || idle=
[ "$type" = integer ] && [ "$idle" = 7200 ] || {
    echo "::error::the current host's screen saver idleTime is '$idle' ($type), not 7200 (integer)"
    fail=1
}
exit "$fail"
