#!/bin/bash
# mise bootstrap (28-bootstrap.sh) wrote macos/roles/keyboard's preferences and units. The later
# plans and statuses say that each is in sync. This case reads some back, as defaults and plutil
# print them, so that a hotkey, a mapping or an array of another type or value has a name here.

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

# The layout's files are root's at 644 and equal the role's, byte for byte.
bundle='/Library/Keyboard Layouts/KeyboardLayout.bundle'
for file in Contents/Info.plist Contents/version.plist Contents/Resources/ENG.keylayout \
    Contents/Resources/RUS.keylayout; do
    cmp -s "$bundle/$file" "macos/roles/keyboard/files/KeyboardLayout.bundle/$file" || {
        echo "::error::$bundle/$file is not the role's file"
        fail=1
    }
    owner=$(stat -f '%Su %Lp' "$bundle/$file" 2>/dev/null) || owner=
    [ "$owner" = "root 644" ] || {
        echo "::error::$bundle/$file is '$owner', not root 644"
        fail=1
    }
done

# input_source DOMAIN KEYPATH WANT [TYPE]: the value at KEYPATH of a preference, and its type,
# which is string unless TYPE says another.
input_source() {
    local plist got type
    plist=$(defaults export "$1" -) || plist=
    got=$(plutil -extract "$2" raw -o - - <<<"$plist" 2>/dev/null) || got=
    type=$(plutil -type "$2" - <<<"$plist" 2>/dev/null) || type=
    if [ "$got" != "$3" ] || [ "$type" != "${4:-string}" ]; then
        echo "::error::$1 $2 is '$got' ($type), want '$3' (${4:-string})"
        fail=1
    fi
}

# The two layouts, each by its ID and name, and no layout of Apple's.
ids=(-20611 -21268)
names=(ENG RUS)
for i in 0 1; do
    input_source com.apple.inputsources "AppleEnabledThirdPartyInputSources.$i.KeyboardLayout ID" "${ids[i]}" integer
    input_source com.apple.inputsources "AppleEnabledThirdPartyInputSources.$i.KeyboardLayout Name" "${names[i]}"
    input_source com.apple.inputsources "AppleEnabledThirdPartyInputSources.$i.InputSourceKind" "Keyboard Layout"
done
input_source com.apple.HIToolbox "AppleEnabledInputSources" 2 array
input_source com.apple.HIToolbox "AppleEnabledInputSources.0.Bundle ID" com.apple.CharacterPaletteIM
input_source com.apple.HIToolbox "AppleEnabledInputSources.1.Bundle ID" com.apple.PressAndHold
exit "$fail"
