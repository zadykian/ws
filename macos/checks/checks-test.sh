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

# nofail NAME... runs a script's check and succeeds where it prints no FAIL line.
nofail() {
    local out
    out=$(inside "$@" 2>&1)
    ! grep -q '^FAIL' <<<"$out"
}

# ws and mise exit as the variables say.
stub ws 'exit "${PLAN:-0}"'
stub mise 'exit "${STATUS:-0}"'
PLAN=0 expect "a plan that exits 0" ok inside setup.sh plan_zero
PLAN=2 expect "a plan that exits 2" fail inside setup.sh plan_zero
STATUS=1 expect "a status that exits 1" fail inside setup.sh status_zero
expect "a status that exits 0" ok inside setup.sh status_zero

# The roles' files name the formulae, the Dock's apps and the login items.
expect "the formulae are read" ok grep -qx colima <(inside setup.sh formulae)
expect "the Dock's nine apps are read" ok test "$(inside setup.sh dock_apps | wc -l)" -eq 9
expect "the six login items are read" ok test "$(inside setup.sh login_apps | wc -l)" -eq 6
expect "formulae that stand nowhere" fail inside setup.sh missing_formulae
for f in $(inside setup.sh formulae); do mkdir "$tmp/brew/opt/$f"; done
expect "formulae that all stand" ok inside setup.sh missing_formulae
expect "apps that stand nowhere" fail inside setup.sh missing_apps

# defaults prints the Dock's apps URL-encoded, a pair of lines each.
inside setup.sh dock_apps | while IFS= read -r app; do
    printf '                "_CFURLString" = "file://%s/";\n' "$(printf '%s' "$app" | sed 's/ /%20/g')"
    printf '                "_CFURLStringType" = 15;\n'
done >"$tmp/dock"
tac "$tmp/dock" >"$tmp/dock-back"
stub defaults 'cat "$DOCK"'
DOCK=$tmp/dock expect "the Dock in order" ok inside setup.sh dock_order
DOCK=$tmp/dock-back expect "the Dock in another order" fail inside setup.sh dock_order

go_want=$(sed -n 's/^go = "\([^"]*\)".*/\1/p' "$dir/../roles/devtools/mise.macos.toml")
stub zsh 'case "$1" in -lc) echo "go version go'"$go_want"' darwin/arm64" ;; *) cat "$ZSH_OUT" ;; esac'
expect "the lock's Go in a login shell" ok inside setup.sh go_of_mise
stub zsh 'echo "go version go0.0.1 darwin/arm64"'
expect "another Go in a login shell" fail inside setup.sh go_of_mise

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
BTM
count() { inside setup.sh btm_count "$1" "$(cat "$tmp/btm")"; }
expect "one login item of Discord" ok test "$(count Discord.app)" -eq 1
expect "two login items of Claude" ok test "$(count Claude.app)" -eq 2
expect "no login item of Mail" ok test "$(count Mail.app)" -eq 0

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
expect "one login item of Rectangle Pro" ok test "$(inside setup.sh btm_count "Rectangle Pro.app" "$(cat "$tmp/btm-once")")" -eq 1

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
{
    cat "$tmp/zsh-ok"
    echo "Insecure directories: /x"
} >"$tmp/zsh-ask"
ZSH_OUT=$tmp/zsh-ask expect "a shell that asks about insecure directories" fail nofail setup.sh check10
BTM=$tmp/btm-once expect "six login items that open once" ok nofail setup.sh check5
BTM=$tmp/btm-twice expect "a login item that opens twice" fail nofail setup.sh check5
only_claude() { [ "$(inside setup.sh check5 2>&1 | grep '^FAIL 5 .* opens once')" = "FAIL 5 Claude.app opens once at login: 2 records name it as a login item" ]; }
BTM=$tmp/btm-twice expect "only Claude opens twice" ok only_claude

[ "$failed" -eq 0 ] || exit 1
echo "checks-test: ok"
