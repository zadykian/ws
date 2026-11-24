#!/bin/bash
# mise bootstrap (28-bootstrap.sh) wrote macos/roles/keyboard's preferences. The later plans and
# statuses say that each is in sync. This case reads some back, as defaults and plutil print them,
# so that a hotkey or a mapping of another type or value has a name here.

set -eo pipefail

fail=0
hotkeys=$(defaults export com.apple.symbolichotkeys -)

# hotkey ID PATH WANT: the value at AppleSymbolicHotKeys.ID.PATH, as plutil prints it raw.
hotkey() {
    local got
    got=$(plutil -extract "AppleSymbolicHotKeys.$1.$2" raw -o - - <<<"$hotkeys" 2>/dev/null) || got=
    if [ "$got" != "$3" ]; then
        echo "::error::hotkey $1 $2 is '$got', want $3"
        fail=1
    fi
}

hotkey 61 enabled true
hotkey 61 value.type standard
hotkey 61 value.parameters.2 1048576
hotkey 64 enabled true
hotkey 64 value.parameters.2 524288
hotkey 160 enabled true
hotkey 160 value.parameters.0 108
for id in 7 21 60 65 164 190 215 260; do
    hotkey "$id" enabled false
done

# Caps Lock is Control, in the current host's domain.
map=$(defaults -currentHost read NSGlobalDomain com.apple.keyboard.modifiermapping.0-0-0 2>/dev/null) || map=
for want in 'ModifierMappingSrc = 30064771129;' 'ModifierMappingDst = 30064771300;'; do
    grep -q "$want" <<<"$map" || {
        echo "::error::the Caps Lock mapping lacks '$want'"
        fail=1
    }
done

# The indicator's value is the string 0: mise compares the type.
type=$(defaults read-type NSGlobalDomain TSMLanguageIndicatorEnabled 2>/dev/null | sed 's/^Type is //') || type=
value=$(defaults read NSGlobalDomain TSMLanguageIndicatorEnabled 2>/dev/null) || value=
if [ "$type" != string ] || [ "$value" != 0 ]; then
    echo "::error::TSMLanguageIndicatorEnabled is $value ($type), want 0 (string)"
    fail=1
fi
exit "$fail"
