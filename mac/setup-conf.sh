# shellcheck shell=sh
# The functions of mac/setup.sh that read ~/.config/ws/mac.conf, the Mac's settings. setup-lib.sh
# sources it, so that a script that sources setup-lib.sh has them too. They have ws conf check the
# file, and set a variable named for each of mac.conf's keys.

# Secretive's agent: its socket, and a .pub in PublicKeys for each key it holds (Secretive 4.0.0).
# shellcheck disable=SC2034 # for setup-identity.sh and setup-git.sh
ws_secretive=$HOME/Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data

# ws_conf FILE reads the Mac's settings from FILE, ~/.config/ws/mac.conf, as ws conf checks them,
# and sets a variable for each key. It evaluates ws conf's quoted assignments, never the file, which
# stays unrun. It exits where ws conf refuses FILE, whose message says why.
ws_conf() {
    ws_conf_eval "$1"
}

# ws_conf_lines FILE is ws_conf's reading of FILE with each line alone, so that it checks a line
# without the keys that go with it.
ws_conf_lines() {
    ws_conf_eval --lines "$1"
}

# ws_conf_eval [--lines] FILE evaluates what ws conf prints for FILE, and exits where it fails.
ws_conf_eval() {
    ws_conf_out=$(ws_conf_run "$@") || exit 1
    eval "$ws_conf_out"
}

# ws_lan_host CIDR ADDRESS succeeds where ws conf takes ADDRESS as a host's in the network CIDR.
ws_lan_host() {
    printf 'WS_LAN=%s\nWS_LAN_ADDRESS=%s\n' "$1" "$2" | ws_conf_run /dev/stdin >/dev/null 2>&1
}

# ws_conf_run ARGS... runs ws conf with ARGS in a ws built from this clone, so that mac.conf is
# checked by the rules of the code being run, not by those of an earlier build in ~/.local/bin. The
# build is in the clone's .cache, and is made again only where a Go file or go.mod of mac/ is newer
# than it, so that a check costs no build in the common case and --check writes no file of the
# Mac's. WS_BIN, where set, is the ws to run, built or not.
# shellcheck disable=SC2154 # dir is the directory of setup.sh or wizard.sh, which sources this
ws_conf_run() {
    ws_conf_bin=${WS_BIN:-${dir%/*}/.cache/ws-conf/ws}
    if [ -z "${WS_BIN:-}" ]; then
        if ! command -v ws_build >/dev/null 2>&1; then
            # shellcheck source=/dev/null # build.sh is in mac/, as ws_build is
            . "$dir/build.sh"
        fi
        if [ ! -x "$ws_conf_bin" ] ||
            [ -n "$(find "$dir" \( -name '*.go' -o -name go.mod -o -name go.sum \) -newer "$ws_conf_bin" 2>/dev/null | head -n 1)" ]; then
            ws_build "${dir%/*}" "$ws_conf_bin" >&2 || ws_die "cannot build ws"
        fi
    fi
    "$ws_conf_bin" conf "$@"
}
