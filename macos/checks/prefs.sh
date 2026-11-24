#!/bin/bash
# The real Mac's checks of its preferences: macos/checks/README.md. Run as the Mac's user as
#
#   prefs.sh [CHECK...]
#
# CHECK is a trial's number on the page. 2 is the preferences' types, 3 the layout, 4 the hotkeys,
# 6 the updates and the display, and 7 the wallpaper. Without one, it runs each. It prints a line a
# check, ok or FAIL, and exits 1 where any fails. It changes nothing.
set -uo pipefail

dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=macos/checks/lib.sh
. "$dir/lib.sh"

keyboard=$tree/macos/roles/keyboard/mise.macos.toml

# pref DOMAIN KEY TYPE VALUE succeeds where defaults read-type prints TYPE and the value is VALUE.
pref() {
    local type value
    type=$(defaults read-type "$1" "$2" 2>/dev/null | sed 's/^Type is //') || type=
    value=$(defaults read "$1" "$2" 2>/dev/null) || value=
    [ "$type" = "$3" ] || {
        echo "$1 $2 is of type '$type', not $3"
        return 1
    }
    awk -v a="$value" -v b="$4" -v t="$3" \
        'BEGIN { exit !((t == "float" || t == "integer") ? a + 0 == b + 0 : a == b) }' || {
        echo "$1 $2 is $value, not $4"
        return 1
    }
}

# types is each preference that desktop and keyboard write: DOMAIN|KEY|TYPE|VALUE, the type as
# defaults read-type prints it. A bool reads as 1 or 0.
types() {
    cat <<'EOT'
com.apple.dock|tilesize|integer|74
com.apple.dock|show-recents|boolean|0
com.apple.dock|minimize-to-application|boolean|1
com.apple.dock|wvous-br-corner|integer|1
com.apple.finder|FXPreferredViewStyle|string|clmv
com.apple.finder|NewWindowTarget|string|PfHm
com.apple.finder|ShowRecentTags|boolean|0
com.apple.assistant.support|Assistant Enabled|boolean|0
com.apple.screencapture|target|string|clipboard
com.apple.screencapture|target-screenshot|string|clipboard
com.apple.screencapture|target-screenrecording|string|file
com.apple.screencapture|style|string|selection
com.apple.WindowManager|EnableStandardClickToShowDesktop|boolean|0
com.apple.WindowManager|EnableTiledWindowMargins|boolean|0
com.apple.WindowManager|EnableTilingByEdgeDrag|boolean|0
com.apple.WindowManager|EnableTopTilingByEdgeDrag|boolean|0
com.apple.WindowManager|EnableTilingOptionAccelerator|boolean|0
com.apple.WindowManager|StandardHideWidgets|boolean|1
com.apple.WindowManager|StageManagerHideWidgets|boolean|1
NSGlobalDomain|AppleInterfaceStyle|string|Dark
NSGlobalDomain|AppleInterfaceStyleSwitchesAutomatically|boolean|0
NSGlobalDomain|AppleShowAllExtensions|boolean|1
NSGlobalDomain|AppleShowScrollBars|string|Always
NSGlobalDomain|AppleEnableSwipeNavigateWithScrolls|boolean|0
NSGlobalDomain|NSGlassTintAmount|float|1
NSGlobalDomain|com.apple.trackpad.scaling|float|0.5
NSGlobalDomain|com.apple.sound.beep.volume|float|0
NSGlobalDomain|com.apple.sound.beep.sound|string|/System/Library/Sounds/Tink.aiff
NSGlobalDomain|com.apple.sound.uiaudio.enabled|integer|0
NSGlobalDomain|TSMLanguageIndicatorEnabled|string|0
EOT
}

# in_doubt lists the preferences whose domain or type the review doubted.
in_doubt() {
    printf '%s\n' \
        'NSGlobalDomain com.apple.sound.uiaudio.enabled' \
        'com.apple.screencapture target-screenshot' \
        'com.apple.screencapture target-screenrecording' \
        'NSGlobalDomain NSGlassTintAmount'
}

check2() {
    local domain key type value
    while IFS='|' read -r domain key type value; do
        check "2 $domain $key is $type" pref "$domain" "$key" "$type" "$value"
    done < <(types)
    while read -r domain key; do
        info "$domain $key: $(defaults read-type "$domain" "$key" 2>&1), $(defaults read "$domain" "$key" 2>&1)"
    done < <(in_doubt)
    check "2 the current host's screen saver delay is 7200" test "$(defaults -currentHost read com.apple.screensaver idleTime 2>&1)" = 7200
    check "2 mise's status finds the preferences in sync" status_zero
}

check3() {
    waits "3 the layout and the input sources wait for the bundle's files, under #1536"
    info "input source: $(defaults read com.apple.HIToolbox AppleCurrentKeyboardLayoutInputSourceID 2>&1)"
    check "3 setup's plan still exits 0" plan_zero
}

# hotkey ID PATH WANT succeeds where the value at AppleSymbolicHotKeys.ID.PATH reads WANT.
hotkey() {
    local got
    got=$(plutil -extract "AppleSymbolicHotKeys.$1.$2" raw -o - - <<<"$hotkeys" 2>/dev/null) || got=
    [ "$got" = "$3" ] || {
        echo "hotkey $1 $2 is '$got', not $3"
        return 1
    }
}

# hotkey_bool ID succeeds where the hotkey's enabled is a boolean in the plist, as mise writes it.
hotkey_bool() {
    local got
    got=$(plutil -type "AppleSymbolicHotKeys.$1.enabled" - <<<"$hotkeys" 2>/dev/null) || got=
    [ "$got" = bool ] || {
        echo "hotkey $1 enabled is '$got', not bool"
        return 1
    }
}

# capslock succeeds where the current host maps Caps Lock to Control on the built-in keyboard.
capslock() {
    local map want
    map=$(defaults -currentHost read NSGlobalDomain com.apple.keyboard.modifiermapping.0-0-0 2>&1)
    for want in 'ModifierMappingSrc = 30064771129;' 'ModifierMappingDst = 30064771300;'; do
        grep -q "$want" <<<"$map" || {
            echo "the Caps Lock mapping lacks '$want'"
            return 1
        }
    done
}

# hotkey_on ID CODE KEY MODIFIERS succeeds where hotkey ID is on at that key.
hotkey_on() {
    hotkey "$1" enabled true && hotkey "$1" value.type standard &&
        hotkey "$1" value.parameters.0 "$2" && hotkey "$1" value.parameters.1 "$3" &&
        hotkey "$1" value.parameters.2 "$4"
}

# off_ids prints the ID of each hotkey that keyboard turns off.
off_ids() { sed -n 's/^path = \["\([0-9]*\)", "enabled"\]$/\1/p' "$keyboard"; }

check4() {
    local id lacked=
    hotkeys=$(defaults export com.apple.symbolichotkeys - 2>&1)
    check "4 Command+Space is on, for the next input source" hotkey_on 61 32 49 1048576
    check "4 Option+Space is on, for Spotlight" hotkey_on 64 32 49 524288
    check "4 Command+L is on, hotkey 160" hotkey_on 160 108 37 1048576
    check "4 enabled is a boolean" hotkey_bool 61
    check "4 Caps Lock acts as Control" capslock
    for id in $(off_ids); do
        check "4 hotkey $id is off" hotkey "$id" enabled false
        plutil -extract "AppleSymbolicHotKeys.$id.value" raw -o - - <<<"$hotkeys" >/dev/null 2>&1 || lacked="$lacked $id"
    done
    info "off IDs that macOS lacked, which mise made:${lacked:- none}"
    info "34 and 98 stay as macOS has them: $(plutil -extract AppleSymbolicHotKeys.34.enabled raw -o - - <<<"$hotkeys" 2>&1), $(plutil -extract AppleSymbolicHotKeys.98.enabled raw -o - - <<<"$hotkeys" 2>&1)"
}

# displaysleep0 succeeds where pmset's list shows displaysleep 0 under the mains' heading.
displaysleep0() {
    pmset -g custom | awk '/^AC Power:/ {ac = 1; next} /^[^ ]/ {ac = 0} ac && $1 == "displaysleep" && $2 == 0 {found = 1} END {exit !found}'
}

check6() {
    local key
    check "6 the display stays on while on power" displaysleep0
    check "6 the App Store's AutoUpdate is 0" test "$(defaults read /Library/Preferences/com.apple.commerce AutoUpdate 2>&1)" = 0
    for key in AutomaticCheckEnabled AutomaticDownload CriticalUpdateInstall AutomaticallyInstallMacOSUpdates AutomaticallyInstallAppUpdates; do
        info "com.apple.SoftwareUpdate $key: $(defaults read /Library/Preferences/com.apple.SoftwareUpdate "$key" 2>&1)"
    done
    info "com.apple.commerce AutoUpdateRestartRequired: $(defaults read /Library/Preferences/com.apple.commerce AutoUpdateRestartRequired 2>&1)"
}

# stone succeeds where wallpaper prints a line a screen and each is Stone.
stone() {
    local out
    out=$(wallpaper get) || return 1
    [ -n "$out" ] || {
        echo "wallpaper get prints no screen"
        return 1
    }
    printf '%s\n' "$out" | awk '$0 != "/System/Library/Desktop Pictures/Solid Colors/Stone.png" { bad = 1; print "not Stone: " $0 } END { exit bad }'
}

check7() {
    check "7 each screen of this Space shows Stone" stone
    info "screens: $(wallpaper get 2>&1 | grep -c .)"
}

main() {
    local arg all=(2 3 4 6 7)
    [ $# -gt 0 ] || set -- "${all[@]}"
    for arg in "$@"; do
        case $arg in
        2) check2 ;;
        3) check3 ;;
        4) check4 ;;
        6) check6 ;;
        7) check7 ;;
        *)
            echo "prefs.sh: no check $arg here" >&2
            exit 2
            ;;
        esac
    done
    finish
}

# checks-test.sh sources the file for its functions.
[ "${BASH_SOURCE[0]}" != "$0" ] || main "$@"
