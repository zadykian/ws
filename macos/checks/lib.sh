# shellcheck shell=bash
# What the Mac's checks share: macos/checks/README.md. A script sources it and runs a check a
# function. A check prints ok or FAIL, and changes nothing on the Mac.

fails=0
ws=${WS_BIN:-/usr/local/bin/ws}
mise=${MISE_BIN:-/usr/local/bin/mise}
# tree is ws's tree on the Mac, which holds the roles' files that the checks read.
tree=${WS_TREE:-/opt/ws}
# mise pours the formulae into Homebrew's prefix, and a login shell finds them there.
brew=${WS_BREW:-/opt/homebrew}
PATH=$brew/bin:$PATH

pass() { printf 'ok   %s\n' "$1"; }
fail() {
    printf 'FAIL %s: %s\n' "$1" "$2"
    fails=$((fails + 1))
}
info() { printf 'info %s\n' "$*"; }
# waits says that a check waits for a build that is not here yet. It fails nothing.
waits() { printf 'wait %s\n' "$*"; }

# check runs the command and prints ok where it succeeds, else FAIL with the last line it printed.
check() {
    local name=$1 out
    shift
    if out=$("$@" 2>&1); then pass "$name"; else fail "$name" "$(printf '%s' "$out" | tail -n 1)"; fi
}

# plan_zero succeeds where setup's plan, in both envs, exits 0.
plan_zero() {
    local status=0
    "$ws" macos setup --plan >/dev/null 2>&1 || status=$?
    [ "$status" -eq 0 ] || echo "ws macos setup --plan exits $status, not 0"
    return "$status"
}

# status_zero succeeds where mise finds nothing missing in the first run's env and the apps'.
status_zero() {
    local out
    out=$("$mise" -C "$tree/macos" -E ws,apps bootstrap status --missing 2>&1) || {
        printf '%s\n' "$out" | tail -n 1
        return 1
    }
}

# apps_of FILE HEADER prints the apps of a table of mise's settings, a path a line.
apps_of() {
    awk -v h="[$2]" '
        $0 == h { on = 1; next }
        /^\[/ { on = 0 }
        on && /^ *"\/.*\.app",$/ { gsub(/^ *"|",$/, ""); print }
    ' "$1"
}

apps_file=$tree/macos/roles/apps/mise.apps.toml
dock_apps() { apps_of "$apps_file" bootstrap.settings.macos.dock; }
login_apps() { apps_of "$apps_file" bootstrap.settings.macos.login_items; }

finish() { [ "$fails" -eq 0 ]; }
