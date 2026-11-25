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

# layout_src is the role's copy of the layout, and bundle is where the Mac keeps it.
layout_src=$tree/macos/roles/keyboard/files/KeyboardLayout.bundle
bundle=${WS_LAYOUT:-/Library/Keyboard Layouts/KeyboardLayout.bundle}

# layout_file FILE succeeds where the Mac's FILE of the bundle is root's at 644 and the role's.
layout_file() {
    local owner
    cmp -s "$bundle/$1" "$layout_src/$1" || {
        echo "$bundle/$1 is not the role's file"
        return 1
    }
    owner=$(stat -f '%Su %Lp' "$bundle/$1" 2>&1)
    [ "$owner" = "root 644" ] || {
        echo "$bundle/$1 is '$owner', not root 644"
        return 1
    }
}

# layout_alone succeeds where the bundle holds the role's files and no other.
layout_alone() {
    [ "$(cd "$bundle" && find . -type f | sort)" = "$(cd "$layout_src" && find . -type f | sort)" ] || {
        echo "the bundle's files differ from the role's"
        return 1
    }
}

# input_source DOMAIN KEYPATH WANT [TYPE] succeeds where the preference at KEYPATH reads WANT, of
# the type string unless TYPE says another.
input_source() {
    local plist got type
    plist=$(defaults export "$1" - 2>&1) || plist=
    got=$(plutil -extract "$2" raw -o - - <<<"$plist" 2>/dev/null) || got=
    type=$(plutil -type "$2" - <<<"$plist" 2>/dev/null) || type=
    [ "$got" = "$3" ] && [ "$type" = "${4:-string}" ] || {
        echo "$1 $2 is '$got' ($type), not '$3' (${4:-string})"
        return 1
    }
}

# in_use succeeds where the input source in use is one of the layout's two, by the IDs of its plist.
in_use() {
    local got
    got=$(defaults read com.apple.HIToolbox AppleCurrentKeyboardLayoutInputSourceID 2>&1)
    grep -A1 '<key>TISInputSourceID</key>' "$layout_src/Contents/Info.plist" |
        sed -n 's/.*<string>\(.*\)<\/string>.*/\1/p' | grep -qxF "$got" || {
        echo "the input source in use is '$got', not one of the layout's"
        return 1
    }
}

check3() {
    local file i ids=(-20611 -21268) names=(ENG RUS)
    for file in $(cd "$layout_src" && find . -type f | sort); do
        check "3 $file is the role's, root's at 644" layout_file "$file"
    done
    check "3 the bundle holds the role's files alone" layout_alone
    for i in 0 1; do
        check "3 input source $i of the layout is ${names[i]}" input_source com.apple.inputsources \
            "AppleEnabledThirdPartyInputSources.$i.KeyboardLayout ID" "${ids[i]}" integer
        check "3 input source $i is named ${names[i]}" input_source com.apple.inputsources \
            "AppleEnabledThirdPartyInputSources.$i.KeyboardLayout Name" "${names[i]}"
    done
    check "3 HIToolbox enables two sources, no layout of Apple's" input_source com.apple.HIToolbox \
        AppleEnabledInputSources 2 array
    check "3 the input source in use is the layout's" in_use
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
