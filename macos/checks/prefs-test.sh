# shellcheck shell=bash
# The cases of prefs.sh's functions. checks-test.sh sources the file, with its temp dir, its
# stand-ins and expect, so that each file stays under the size cap.
# shellcheck disable=SC2016 # each stand-in's body expands its variables when it runs, not here
# shellcheck disable=SC2154 # tmp and dir are checks-test.sh's, which sources the file

# The preferences' types come from defaults read-type, as the real tool prints them.
cat >"$tmp/bin/defaults" <<'DEF'
#!/bin/sh
case "$1" in
-currentHost) echo 7200 ;;
read-type) echo "Type is $(grep -F "|$3|" "$TYPES" | head -n 1 | cut -d'|' -f3)" ;;
read) grep -F "|$3|" "$TYPES" | head -n 1 | cut -d'|' -f4 ;;
esac
DEF
inside prefs.sh types >"$tmp/types"
TYPES=$tmp/types expect "preferences of the types the roles write" ok nofail prefs.sh check2
sed 's/|tilesize|integer|/|tilesize|float|/' "$tmp/types" >"$tmp/types-float"
TYPES=$tmp/types-float expect "preferences with one of another type" fail nofail prefs.sh check2
TYPES=$tmp/types-float expect "a tilesize of another type" fail inside prefs.sh pref com.apple.dock tilesize integer 74
TYPES=$tmp/types expect "a mismatched value" fail inside prefs.sh pref com.apple.dock tilesize integer 75

# Each key of the table, but the Dock's and Finder's, is named in a role's file.
for key in $(inside prefs.sh types | awk -F'|' '$1 != "com.apple.dock" && $1 != "com.apple.finder" && $2 !~ / / { print $2 }'); do
    expect "$key is written by a role" ok grep -qF "$key" "$dir/../roles/desktop/mise.macos.toml" "$dir/../roles/keyboard/mise.macos.toml"
done

# plutil -extract prints one value of the hotkeys' plist, and fails where there is none.
stub plutil 'k=$2; [ "$1" = -type ] && k=type:$2; grep -m1 "^$k=" "$PLUTIL_DB" | cut -d= -f2- | grep .'
{
    echo AppleSymbolicHotKeys.61.enabled=true
    echo AppleSymbolicHotKeys.61.value.type=standard
    echo AppleSymbolicHotKeys.61.value.parameters.0=32
    echo AppleSymbolicHotKeys.61.value.parameters.1=49
    echo AppleSymbolicHotKeys.61.value.parameters.2=1048576
    echo AppleSymbolicHotKeys.7.enabled=false
    echo type:AppleSymbolicHotKeys.61.enabled=bool
    echo type:AppleSymbolicHotKeys.7.enabled=string
} >"$tmp/plist"
on61() { hotkeys=x hotkey_on 61 32 49 1048576; }
on61_wrong() { hotkeys=x hotkey_on 61 32 49 524288; }
off7() { hotkeys=x hotkey 7 enabled false; }
bool61() { hotkeys=x hotkey_bool 61; }
bool7() { hotkeys=x hotkey_bool 7; }
off61() { hotkeys=x hotkey 61 enabled false; }
PLUTIL_DB=$tmp/plist expect "hotkey 61 on at Command+Space" ok inside prefs.sh on61
PLUTIL_DB=$tmp/plist expect "hotkey 61 on at another modifier" fail inside prefs.sh on61_wrong
PLUTIL_DB=$tmp/plist expect "hotkey 7 off" ok inside prefs.sh off7
PLUTIL_DB=$tmp/plist expect "hotkey 61 not off" fail inside prefs.sh off61
PLUTIL_DB=$tmp/plist expect "an enabled that is a boolean" ok inside prefs.sh bool61
PLUTIL_DB=$tmp/plist expect "an enabled that is a string" fail inside prefs.sh bool7
expect "the 63 off hotkeys are read" ok test "$(inside prefs.sh off_ids | wc -l)" -eq 63

stub defaults 'cat "$MAP"'
printf '{\n    ModifierMappingDst = 30064771300;\n    ModifierMappingSrc = 30064771129;\n}\n' >"$tmp/map"
printf '{\n    ModifierMappingDst = 30064771300;\n}\n' >"$tmp/map-half"
MAP=$tmp/map expect "Caps Lock as Control" ok inside prefs.sh capslock
MAP=$tmp/map-half expect "a mapping with no source" fail inside prefs.sh capslock

# check4 reads each hotkey and its keys from keyboard's file. The plist holds them as mise writes.
on_read() { inside prefs.sh hotkey_entries | awk '$2 == "on" && NF == 5 && $3 $4 $5 ~ /^[0-9]+$/' | wc -l; }
expect "the three on hotkeys are read with their keys" ok test "$(on_read)" -eq 3
inside prefs.sh hotkey_entries | while read -r id state code key mods; do
    k=AppleSymbolicHotKeys.$id
    if [ "$state" = on ]; then
        printf '%s\n' "$k.enabled=true" "$k.value.type=standard" "$k.value.parameters.0=$code" \
            "$k.value.parameters.1=$key" "$k.value.parameters.2=$mods" "type:$k.enabled=bool"
    else
        printf '%s\n' "$k.enabled=false" "$k.value=x"
    fi
done >"$tmp/hotkeys"
sed 's/^\(AppleSymbolicHotKeys\.64\.value\.parameters\.2=\).*/\10/' "$tmp/hotkeys" >"$tmp/hotkeys-64"
MAP=$tmp/map PLUTIL_DB=$tmp/hotkeys expect "the hotkeys as keyboard writes them" ok nofail prefs.sh check4
MAP=$tmp/map PLUTIL_DB=$tmp/hotkeys-64 expect "hotkey 64 at other keys" fail nofail prefs.sh check4

# The layout's files are the role's, root's at 644. stat prints the owner and mode that OWNER says.
cp -R "$dir/../roles/keyboard/files/ws.bundle" "$tmp/layout"
export WS_LAYOUT=$tmp/layout
stub stat 'echo "${OWNER:-root 644}"'
expect "a file of the layout as the role has it" ok inside prefs.sh layout_file ./Contents/Info.plist
OWNER="v 644" expect "a file of the layout that v owns" fail inside prefs.sh layout_file ./Contents/Info.plist
OWNER="root 600" expect "a file of the layout at 600" fail inside prefs.sh layout_file ./Contents/Info.plist
expect "a bundle of the role's files alone" ok inside prefs.sh layout_alone
printf x >>"$tmp/layout/Contents/Info.plist"
expect "a file that differs from the role's" fail inside prefs.sh layout_file ./Contents/Info.plist
cp "$dir/../roles/keyboard/files/ws.bundle/Contents/Info.plist" "$tmp/layout/Contents/Info.plist"
printf x >"$tmp/layout/Contents/extra"
expect "a bundle with a file more" fail inside prefs.sh layout_alone
rm "$tmp/layout/Contents/extra" "$tmp/layout/Contents/version.plist"
expect "a bundle with a file less" fail inside prefs.sh layout_alone

# plutil prints the input sources' values and types, and defaults the one in use.
stub defaults 'case "$1" in export) echo plist ;; read) echo "$IN_USE" ;; esac'
{
    echo "AppleEnabledThirdPartyInputSources.0.KeyboardLayout ID=-20611"
    echo "type:AppleEnabledThirdPartyInputSources.0.KeyboardLayout ID=integer"
    echo "AppleEnabledThirdPartyInputSources.1.KeyboardLayout Name=RUS"
    echo "type:AppleEnabledThirdPartyInputSources.1.KeyboardLayout Name=integer"
    echo "AppleEnabledInputSources=4"
    echo "type:AppleEnabledInputSources=array"
} >"$tmp/sources"
sources() { PLUTIL_DB=$tmp/sources inside prefs.sh input_source "$@"; }
expect "an input source of the ID and type" ok sources com.apple.inputsources \
    "AppleEnabledThirdPartyInputSources.0.KeyboardLayout ID" -20611 integer
expect "an input source of another ID" fail sources com.apple.inputsources \
    "AppleEnabledThirdPartyInputSources.0.KeyboardLayout ID" -21268 integer
expect "an input source ID that is a string" fail sources com.apple.inputsources \
    "AppleEnabledThirdPartyInputSources.0.KeyboardLayout ID" -20611
expect "a name that is the wrong type" fail sources com.apple.inputsources \
    "AppleEnabledThirdPartyInputSources.1.KeyboardLayout Name" RUS
expect "four enabled sources" ok sources com.apple.HIToolbox AppleEnabledInputSources 4 array
expect "two enabled sources" fail sources com.apple.HIToolbox AppleEnabledInputSources 2 array
layout_id=$(grep -A1 '<key>TISInputSourceID</key>' "$dir/../roles/keyboard/files/ws.bundle/Contents/Info.plist" |
    sed -n 's/.*<string>\(.*\)<\/string>.*/\1/p' | tail -n 1)
IN_USE=$layout_id expect "the layout's input source in use" ok inside prefs.sh in_use
IN_USE=com.apple.keylayout.ABC expect "an input source of Apple's in use" fail inside prefs.sh in_use

# pmset prints a source's settings under its own heading.
printf 'Battery Power:\n displaysleep 2\nAC Power:\n displaysleep 0\n sleep 0\n' >"$tmp/pmset-on"
printf 'Battery Power:\n displaysleep 0\nAC Power:\n displaysleep 10\n' >"$tmp/pmset-off"
stub pmset 'cat "$PMSET"'
PMSET=$tmp/pmset-on expect "a display that stays on with power" ok inside prefs.sh displaysleep0
PMSET=$tmp/pmset-off expect "a display that sleeps with power" fail inside prefs.sh displaysleep0

stone="/System/Library/Desktop Pictures/Solid Colors/Stone.png"
printf '%s\n%s\n' "$stone" "$stone" >"$tmp/wp-ok"
printf '%s\n/Users/x/other.png\n' "$stone" >"$tmp/wp-bad"
: >"$tmp/wp-none"
stub wallpaper 'cat "$WP"'
WP=$tmp/wp-ok expect "two screens of Stone" ok inside prefs.sh stone
WP=$tmp/wp-bad expect "a second screen of another picture" fail inside prefs.sh stone
WP=$tmp/wp-none expect "no screen" fail inside prefs.sh stone
