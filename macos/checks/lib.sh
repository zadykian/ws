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
# waits says that a check needs a value or a step by hand first. It fails nothing.
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

# toml_strings FILE TABLE KEY prints each string of the array KEY in TABLE, a line each. It reads
# the array by its brackets, quotes and comments, not its lines, as tombi may lay it out either way.
toml_strings() {
    awk -v table="[$2]" -v key="$3" '
        function bare(s) { sub(/#.*/, "", s); gsub(/[ \t"]/, "", s); return s }
        function scan(s,   i, c) {
            for (i = 1; i <= length(s); i++) {
                c = substr(s, i, 1)
                if (q != "") {
                    if (q == "\"" && c == "\\") { i++; c = substr(s, i, 1) }
                    else if (c == q) { print str; q = ""; continue }
                    str = str c
                } else if (c == "\"" || c == "'\''") { q = c; str = "" }
                else if (c == "#") return
                else if (c == "[") depth++
                else if (c == "]" && --depth == 0) return
            }
        }
        BEGIN { table = bare(table) }
        depth > 0 { scan($0); next }
        /^[ \t]*\[/ { on = bare($0) == table; next }
        on && index($0, "=") {
            eq = index($0, "=")
            if (bare(substr($0, 1, eq - 1)) == key) scan(substr($0, eq + 1))
        }
    ' "$1"
}

apps_file=$tree/macos/roles/apps/mise.apps.toml
dock_apps() { toml_strings "$apps_file" bootstrap.settings.macos.dock apps; }
login_apps() { toml_strings "$apps_file" bootstrap.settings.macos.login_items apps; }

keyboard=$tree/macos/roles/keyboard/mise.macos.toml

# hotkey_entries prints keyboard's hotkeys, a line each. "ID on CODE KEY MODIFIERS" is one that it
# turns on at a key, and "ID off" one whose enabled it sets false.
hotkey_entries() {
    awk '
        function entry(   p) {
            if (domain != "\"com.apple.symbolichotkeys\"") return
            if (n == 2 && ids[2] == "enabled" && value == "false") print ids[1], "off"
            if (n == 1 && index(value, "enabled = true") && index(value, "parameters = [")) {
                p = substr(value, index(value, "parameters = [") + 14)
                p = substr(p, 1, index(p, "]") - 1)
                gsub(/,/, "", p)
                print ids[1], "on", p
            }
        }
        /^[ \t]*\[/ { entry(); domain = value = ""; n = 0 }
        /^domain *=/ { domain = $3 }
        /^path *=/ {
            p = substr($0, index($0, "[") + 1)
            p = substr(p, 1, index(p, "]") - 1)
            gsub(/[" ]/, "", p)
            n = split(p, ids, ",")
        }
        /^value *=/ { value = substr($0, index($0, "=") + 1); sub(/^ */, "", value) }
        END { entry() }
    ' "$keyboard"
}

# off_ids prints the ID of each hotkey that keyboard turns off.
off_ids() { hotkey_entries | awk '$2 == "off" { print $1 }'; }

finish() { [ "$fails" -eq 0 ]; }
