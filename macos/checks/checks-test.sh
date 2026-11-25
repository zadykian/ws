#!/bin/bash
# Tries the functions of setup.sh, prefs.sh and tools.sh against stand-ins for the Mac's tools.
# The stand-ins print what defaults, plutil, pmset and sfltool print. The test reads the real roles'
# files of the tree. CI's lint job runs it. The scripts themselves run on the Mac alone.
# shellcheck disable=SC2016 # each stand-in's body expands its variables when it runs, not here
set -uo pipefail

dir=${CHECKS_DIR:-$(cd "$(dirname "$0")" && pwd)}
tmp=$(mktemp -d)
mkdir -p "$tmp/bin" "$tmp/brew/opt" "$tmp/home/.gnupg" "$tmp/Caskroom/claude-code" "$tmp/other"
trap 'rm -rf "$tmp"' EXIT
export PATH="$tmp/bin:$PATH" HOME="$tmp/home"
export WS_TREE=${WS_TREE:-$dir/../..} WS_BREW=$tmp/brew WS_BIN=$tmp/bin/ws MISE_BIN=$tmp/bin/mise
failed=0

# expect NAME WANT COMMAND... runs the command. WANT is ok for an exit status of 0, else fail.
expect() {
    local name=$1 want=$2 got=ok
    shift 2
    "$@" >"$tmp/out" 2>&1 || got=fail
    if [ "$got" != "$want" ]; then
        echo "checks-test: $name: wanted $want, got $got" >&2
        sed 's/^/    /' "$tmp/out" >&2
        failed=1
    fi
}

# Each case sources the script, which runs nothing then, in a shell of its own.
inside() {
    local script=$1
    shift
    (
        # shellcheck disable=SC1090 # the path is the test's variable
        . "$dir/$script"
        "$@"
    )
}

# stub NAME BODY writes a program that the scripts find first on the PATH.
stub() { printf '#!/bin/sh\n%s\n' "$2" >"$tmp/bin/$1" && chmod +x "$tmp/bin/$1"; }

# nofail NAME... runs a script's check, and succeeds where it ends with 0, prints an ok line and
# no FAIL line. A check that crashes, or that is not there, exits other than 0.
nofail() {
    local out
    out=$(inside "$@" 2>&1) || return 1
    grep -q '^ok ' <<<"$out" && ! grep -q '^FAIL' <<<"$out"
}
dies() {
    pass "a first check"
    # shellcheck disable=SC2154 # unset on purpose, so that set -u ends the check
    : "$no_such_variable"
}
expect "a check that dies under set -u" fail nofail setup.sh dies
expect "a check that is not there" fail nofail setup.sh check_none

# ws and mise exit as the variables say, and mise prints MISE_OUT's file where one is named.
stub ws 'exit "${PLAN:-0}"'
stub mise '[ -z "${MISE_OUT:-}" ] || cat "$MISE_OUT"; exit "${STATUS:-0}"'
PLAN=0 expect "a plan that exits 0" ok inside setup.sh plan_zero
PLAN=2 expect "a plan that exits 2" fail inside setup.sh plan_zero
STATUS=1 expect "a status that exits 1" fail inside setup.sh status_zero
expect "a status that exits 0" ok inside setup.sh status_zero

# The roles' files name the formulae, the Dock's apps and the login items.
expect "the formulae are read" ok grep -qx colima <(inside setup.sh formulae)
expect "the Dock's nine apps are read" ok test "$(inside setup.sh dock_apps | wc -l)" -eq 9
expect "the seven login items are read" ok test "$(inside setup.sh login_apps | wc -l)" -eq 7
expect "formulae that stand nowhere" fail inside setup.sh missing_formulae
for f in $(inside setup.sh formulae); do mkdir "$tmp/brew/opt/$f"; done
expect "formulae that all stand" ok inside setup.sh missing_formulae
expect "apps that stand nowhere" fail inside setup.sh missing_apps

# A tree of the test's own, whose apps stand under its directory. Its arrays take the layouts that
# TOML allows: on one line, an element on a bracket's line, no last comma, a comment.
mkdir -p "$tmp/tree/macos/roles/apps" "$tmp/tree/macos/roles/devtools" "$tmp/apps/Dock.app" \
    "$tmp/apps/Login Item.app" "$tmp/apps/Quote \"A\".app"
cat >"$tmp/tree/macos/roles/apps/mise.apps.toml" <<APPS
[bootstrap.settings.macos.dock]
other = ["/x.app"]
apps = ["$tmp/apps/Dock.app", "$tmp/apps/Login Item.app"] # a comment, "/y.app"

[ bootstrap.settings.macos.login_items ] # the login items
exclusive = true
apps = [ "$tmp/apps/Login Item.app",
    # "/z.app",
    '$tmp/apps/Quote "A".app',
    "$tmp/apps/Quote \\"A\\".app"
]

[bootstrap.settings.macos.next]
apps = ["/w.app"]
APPS
dock_want=$(printf '%s\n' "$tmp/apps/Dock.app" "$tmp/apps/Login Item.app")
login_want=$(printf '%s\n' "$tmp/apps/Login Item.app" "$tmp/apps/Quote \"A\".app" "$tmp/apps/Quote \"A\".app")
expect "an array on one line" ok test "$(WS_TREE=$tmp/tree inside setup.sh dock_apps)" = "$dock_want"
expect "an array of any layout" ok test "$(WS_TREE=$tmp/tree inside setup.sh login_apps)" = "$login_want"
WS_TREE=$tmp/tree expect "apps that all stand" ok inside setup.sh missing_apps
rmdir "$tmp/apps/Login Item.app"
WS_TREE=$tmp/tree expect "a login item's app that is gone" fail inside setup.sh missing_apps

# defaults prints the Dock's apps URL-encoded, a pair of lines each.
inside setup.sh dock_apps | while IFS= read -r app; do
    printf '                "_CFURLString" = "file://%s/";\n' "$(printf '%s' "$app" | sed 's/ /%20/g')"
    printf '                "_CFURLStringType" = 15;\n'
done >"$tmp/dock"
tac "$tmp/dock" >"$tmp/dock-back"
stub defaults 'cat "$DOCK"'
DOCK=$tmp/dock expect "the Dock in order" ok inside setup.sh dock_order
DOCK=$tmp/dock-back expect "the Dock in another order" fail inside setup.sh dock_order

# The lock's Go, read apart from the script's read of devtools' file, which must find the same.
go_lock=$(awk '/^\[\[tools\.go\]\]/ { on = 1; next } on && /^version/ { gsub(/^[^"]*"|".*$/, ""); print; exit }' \
    "$dir/../mise/mise.macos.lock")
expect "the lock names Go's version" ok test -n "$go_lock"
stub zsh 'case "$1" in -lc) echo "go version go'"$go_lock"' darwin/arm64" ;; *) cat "$ZSH_OUT" ;; esac'
expect "the lock's Go in a login shell" ok inside setup.sh go_of_mise
stub zsh 'echo "go version go0.0.1 darwin/arm64"'
expect "another Go in a login shell" fail inside setup.sh go_of_mise
printf '[tools]\ngo = { version = "%s" }\n' "$go_lock" >"$tmp/tree/macos/roles/devtools/mise.macos.toml"
stub zsh 'echo "go version go darwin/arm64"'
WS_TREE=$tmp/tree expect "a Go pin that the script cannot read" fail inside setup.sh go_of_mise

# sfltool's dump holds a record a login item, and the app's own record apart.
cat >"$tmp/btm" <<'BTM'
 #1:
                 UUID: A
                 Name: Discord
                 Type: login item (0x4)
          Disposition: [enabled, allowed, visible, notified] (0xb)
                  URL: file:///Applications/Discord.app/
 #2:
                 UUID: B
                 Name: Discord
                 Type: app (0x2)
                  URL: file:///Applications/Discord.app/
 #3:
                 UUID: C
                 Name: Claude
                 Type: login item (0x4)
                  URL: file:///Applications/Claude.app/
 #4:
                 UUID: D
                 Name: Claude
                 Type: login item (0x4)
                  URL: file:///Applications/Claude.app/
 #5:
                 UUID: E
                 Name: Stats
                 Type: login item (0x4)
                  URL: file:///Applications/Stats.app/Contents/Library/LoginItems/Launcher.app/
BTM
count() { inside setup.sh btm_count "$1" "$(cat "$tmp/btm")"; }
expect "one login item of Discord" ok test "$(count /Applications/Discord.app)" -eq 1
expect "two login items of Claude" ok test "$(count /Applications/Claude.app)" -eq 2
expect "no login item of Mail" ok test "$(count /System/Applications/Mail.app)" -eq 0
expect "a bundle inside Stats's, apart from Stats" ok test "$(count /Applications/Stats.app)" -eq 0
expect "the bundle inside Stats's" ok test \
    "$(count /Applications/Stats.app/Contents/Library/LoginItems/Launcher.app)" -eq 1

# A file URL writes a space as %20: one record of each login app, Claude's twice in the second dump.
n=0
inside setup.sh login_apps | while IFS= read -r app; do
    n=$((n + 1))
    printf ' #%s:\n Type: login item (0x4)\n URL: file://%s/\n' "$n" "$(printf '%s' "$app" | sed 's/ /%20/g')"
done >"$tmp/btm-once"
{
    cat "$tmp/btm-once"
    printf ' #9:\n Type: login item (0x4)\n URL: file:///Applications/Claude.app/\n'
} >"$tmp/btm-twice"
expect "one login item of Rectangle Pro" ok test "$(inside setup.sh btm_count "/Applications/Rectangle Pro.app" "$(cat "$tmp/btm-once")")" -eq 1

cat >"$tmp/zsh-ok" <<'OUT'
plugins=git zsh-autosuggestions zsh-syntax-highlighting
autosuggestions=1
highlighting=1
update=disabled
OUT
stub zsh 'cat "$ZSH_OUT"'
stub sudo '"$@"'
stub sfltool 'cat "$BTM"'
stub pgrep 'exit 0'
ZSH_OUT=$tmp/zsh-ok expect "a shell that loads the plugins" ok nofail setup.sh check10
sed 's/highlighting=1/highlighting=0/' "$tmp/zsh-ok" >"$tmp/zsh-bad"
ZSH_OUT=$tmp/zsh-bad expect "a shell without highlighting" fail nofail setup.sh check10
# oh-my-zsh's own lines, as its pinned commit prints them.
for ask in "[oh-my-zsh] Insecure completion-dependent directories detected:" \
    "[oh-my-zsh] Would you like to update? [Y/n] " \
    "[oh-my-zsh] It's time to update! You can do that by running \`omz update\`"; do
    printf '%s\n' "$(cat "$tmp/zsh-ok")" "$ask" >"$tmp/zsh-ask"
    ZSH_OUT=$tmp/zsh-ask expect "a shell that prints: $ask" fail nofail setup.sh check10
done

# mise's status prints a row a login item, its state last, with no heading off a terminal.
inside setup.sh login_apps | while IFS= read -r app; do
    printf 'login items  %s    set\n' "$app"
done >"$tmp/status-set"
printf 'packages     brew-cask:spotify  latest  missing\n' >>"$tmp/status-set"
{
    cat "$tmp/status-set"
    printf 'login items  /Applications/Spotify.app    undeclared\n'
} >"$tmp/status-undeclared"
sed 's|^\(login items  /Applications/Claude.app *\)set$|\1missing|' "$tmp/status-set" >"$tmp/status-missing"
grep -v '^login items' "$tmp/status-set" >"$tmp/status-none"
MISE_OUT=$tmp/status-set expect "each login item set" ok inside setup.sh login_status
MISE_OUT=$tmp/status-undeclared expect "an app's own login item" fail inside setup.sh login_status
MISE_OUT=$tmp/status-missing expect "a login item that is missing" fail inside setup.sh login_status
MISE_OUT=$tmp/status-none expect "a status with no login item" fail inside setup.sh login_status
MISE_OUT=$tmp/status-set STATUS=1 expect "a status that fails" fail inside setup.sh login_status

export MISE_OUT=$tmp/status-set
BTM=$tmp/btm-once expect "seven login items that open once" ok nofail setup.sh check5
BTM=$tmp/btm-twice expect "a login item that opens twice" fail nofail setup.sh check5
only_claude() { [ "$(inside setup.sh check5 2>&1 | grep '^FAIL 5 .* opens once')" = "FAIL 5 Claude.app opens once at login: 2 records name it as a login item" ]; }
BTM=$tmp/btm-twice expect "only Claude opens twice" ok only_claude
{
    cat "$tmp/btm-once"
    printf ' #10:\n Type: login item (0x4)\n URL: file:///Applications/Spotify.app/\n'
} >"$tmp/btm-other"
other() { inside setup.sh check5 | grep -qxF "info 5 the dump's login items outside the role: file:///Applications/Spotify.app/;"; }
BTM=$tmp/btm-other expect "an item outside the role, noted" ok other
BTM=$tmp/btm-other expect "an item outside the role, which fails nothing" ok nofail setup.sh check5
unset MISE_OUT

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

# claude resolves into the cask's directory, and its sum is the one before.
printf 'binary' >"$tmp/Caskroom/claude-code/claude"
chmod +x "$tmp/Caskroom/claude-code/claude"
ln -sf "$tmp/Caskroom/claude-code/claude" "$tmp/bin/claude"
stub shasum 'sha256sum "$3"'
sum=$(sha256sum "$tmp/Caskroom/claude-code/claude" | cut -d' ' -f1)
expect "claude in a Caskroom" ok inside tools.sh in_cask
expect "claude's sum is the one before" ok inside tools.sh sum_is "$sum"
expect "claude's sum changed" fail inside tools.sh sum_is 0000
printf 'binary' >"$tmp/other/claude"
chmod +x "$tmp/other/claude"
ln -sf "$tmp/other/claude" "$tmp/bin/claude"
expect "claude outside a Caskroom" fail inside tools.sh in_cask
ln -sf "$tmp/Caskroom/claude-code/claude" "$tmp/bin/claude"
mkdir -p "$HOME/.local/bin"
ln -sf "$tmp/other/claude" "$HOME/.local/bin/claude"
expect "claude of the updater, before the cask's" fail inside tools.sh in_cask

printf 'pinentry-program %s/pinentry-mac\n' "$tmp/bin" >"$HOME/.gnupg/gpg-agent.conf"
expect "a pinentry-mac that is not there" fail inside tools.sh pinentry_named
stub pinentry-mac 'exit 0'
expect "a pinentry-mac that runs" ok inside tools.sh pinentry_named
echo 'pinentry-program /usr/bin/pinentry-curses' >"$HOME/.gnupg/gpg-agent.conf"
expect "another pinentry" fail inside tools.sh pinentry_named

stub socketfilterfw 'echo "Firewall is set to block all non-essential incoming connections: blocking all"'
SOCKETFILTERFW=$tmp/bin/socketfilterfw expect "a firewall that blocks all" ok inside tools.sh blocks_all
stub socketfilterfw 'echo "Block all DISABLED!"'
SOCKETFILTERFW=$tmp/bin/socketfilterfw expect "a firewall that lets connections in" fail inside tools.sh blocks_all
printf 'x' >"$tmp/torrent"
: >"$tmp/torrent-empty"
expect "a downloaded file" ok inside tools.sh downloaded "$tmp/torrent"
expect "an empty file" fail inside tools.sh downloaded "$tmp/torrent-empty"

[ "$failed" -eq 0 ] || exit 1
echo "checks-test: ok"
