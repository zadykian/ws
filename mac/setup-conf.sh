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

# ws_conf FILE reads the Mac's settings from FILE, ~/.config/ws/mac.conf, without running it: a
# KEY=value a line, the value bare or in a pair of quotes, blank lines and lines starting with #
# skipped. It stops on any other line, any other key, a value of the wrong form, and a
# WS_LAN_ADDRESS that is not a host's in WS_LAN. An empty value, KEY= or KEY="", is the key's
# default, as a missing line is, and passes its key's check.
ws_conf_old() {
    [ -f "$1" ] || ws_die "$1 is missing: docs/mac.md says what goes in it"
    ws_conf_lines_old "$1"
    ws_conf_pairs "$1"
}

# ws_conf_lines FILE is ws_conf's reading of FILE, each line alone, with each key's variable set to
# its default first. So it checks a line without the keys that go with it.
# shellcheck disable=SC2154 # ws_cable_net and ws_tunnel_net are lib.sh's
ws_conf_lines_old() {
    WS_USER=root
    WS_TARGET=lan
    WS_PUBLIC_KEY=
    WS_LAN=
    WS_LAN_ADDRESS=
    WS_REMOTE=
    WS_REMOTE_PORT=
    WS_PORT=
    WS_TUNNEL_NETWORKS=
    WS_RESOLVERS=
    WS_CABLE_SERVICE=
    WS_CABLE_PORT=
    WS_GIT_NAME=
    WS_GIT_EMAIL=
    WS_GIT_SIGNING_KEY=
    WS_IDENTITY=
    WS_IDENTITY_AGENT=
    WS_HOOK=
    WS_HOOK_SECRET=
    ws_n=0
    while IFS=' 	' read -r ws_line || [ -n "$ws_line" ]; do
        ws_n=$((ws_n + 1))
        case $ws_line in
        '' | '#'*) continue ;;
        [A-Z]*=*) ;;
        *) ws_die "$1, line $ws_n: not a KEY=value line" ;;
        esac
        ws_key=${ws_line%%=*}
        ws_value=${ws_line#*=}
        case $ws_value in
        \"*\" | \'*\')
            ws_value=${ws_value#?}
            ws_value=${ws_value%?}
            ;;
        esac
        ws_ok=true
        case $ws_key in
        WS_USER)
            # shellcheck disable=SC2034 # for setup-ssh.sh and setup.sh
            WS_USER=${ws_value:-root}
            ws_is_user "$ws_value" || ws_ok=false
            ;;
        WS_TARGET)
            WS_TARGET=${ws_value:-lan}
            case $ws_value in lan | cloud | mac-vm) ;; *) ws_ok=false ;; esac
            ;;
        WS_PUBLIC_KEY)
            # shellcheck disable=SC2034 # for setup-ssh.sh and setup.sh
            WS_PUBLIC_KEY=$ws_value
            ws_is_key "$ws_value" || ws_ok=false
            ;;
        WS_LAN)
            WS_LAN=$ws_value
            ws_is_cidr "$ws_value" && ! ws_cidr_overlaps "$ws_value" "$ws_cable_net" &&
                ! ws_cidr_overlaps "$ws_value" "$ws_tunnel_net" || ws_ok=false
            ;;
        WS_LAN_ADDRESS)
            WS_LAN_ADDRESS=$ws_value
            ws_is_ipv4 "$ws_value" || ws_ok=false
            ;;
        WS_REMOTE)
            WS_REMOTE=$ws_value
            ws_is_domain "$ws_value" || ws_ok=false
            ;;
        WS_REMOTE_PORT)
            WS_REMOTE_PORT=$ws_value
            ws_is_port "$ws_value" || ws_ok=false
            ;;
        WS_PORT)
            # shellcheck disable=SC2034 # for setup.sh
            WS_PORT=$ws_value
            ws_is_port "$ws_value" || ws_ok=false
            ;;
        WS_TUNNEL_NETWORKS)
            # shellcheck disable=SC2034 # for setup.sh
            WS_TUNNEL_NETWORKS=$ws_value
            ws_are_networks "$ws_value" || ws_ok=false
            ;;
        WS_RESOLVERS)
            # shellcheck disable=SC2034 # for setup.sh
            WS_RESOLVERS=$ws_value
            ws_are_resolvers "$ws_value" || ws_ok=false
            ;;
        WS_CABLE_SERVICE)
            # shellcheck disable=SC2034 # for the cable's step
            WS_CABLE_SERVICE=$ws_value
            # A blank needs quotes, as in a shell's assignment; a * first marks a disabled service.
            case $ws_value in
            \** | *[[:cntrl:]]*) ws_ok=false ;;
            *' '*) [ "$ws_value" != "${ws_line#*=}" ] || ws_ok=false ;;
            esac
            ;;
        WS_CABLE_PORT)
            WS_CABLE_PORT=$ws_value
            case $ws_value in *[!A-Za-z0-9_.-]* | -*) ws_ok=false ;; esac
            [ "${#ws_value}" -le 15 ] || ws_ok=false
            ;;
        WS_GIT_NAME)
            # shellcheck disable=SC2034 # for setup-git.sh
            WS_GIT_NAME=$ws_value
            # git's config takes it in quotes, which a " would end and a \ escape.
            case $ws_value in
            *[[:cntrl:]\"\\]*) ws_ok=false ;;
            *' '*) [ "$ws_value" != "${ws_line#*=}" ] || ws_ok=false ;;
            esac
            ;;
        WS_GIT_EMAIL)
            WS_GIT_EMAIL=$ws_value
            case $ws_value in
            *[!A-Za-z0-9._%+@-]* | @*) ws_ok=false ;;
            *@*) ws_is_domain "${ws_value#*@}" || ws_ok=false ;;
            *) ws_ok=false ;;
            esac
            ;;
        WS_GIT_SIGNING_KEY)
            # shellcheck disable=SC2034 # for setup-git.sh
            WS_GIT_SIGNING_KEY=$ws_value
            case ${ws_value#\~} in
            *[!A-Za-z0-9._/@+-]*) ws_ok=false ;;
            /*.pub) ;;
            *) ws_ok=false ;;
            esac
            ;;
        WS_IDENTITY)
            WS_IDENTITY=$ws_value
            case ${ws_value#\~} in
            *[!A-Za-z0-9._/@+-]*) ws_ok=false ;;
            /*.pub) ;;
            *) ws_ok=false ;;
            esac
            ;;
        WS_IDENTITY_AGENT)
            WS_IDENTITY_AGENT=$ws_value
            # An app's socket may hold a blank, which needs quotes, as in a shell's assignment.
            case ${ws_value#\~} in
            *[!A-Za-z0-9._/@+\ -]*) ws_ok=false ;;
            /?*' '*) [ "$ws_value" != "${ws_line#*=}" ] || ws_ok=false ;;
            /?*) ;;
            *) ws_ok=false ;;
            esac
            ;;
        WS_HOOK)
            WS_HOOK=$ws_value
            # OWNER/REPO on GitHub, as ws's /root/repository/REPO: neither side . or .. alone, and
            # REPO not ws, ws's own clone there, in any letter case, which APFS and GitHub ignore.
            case $ws_value in
            *[!A-Za-z0-9._/-]* | */*/* | /* | */ | ./* | ../* | */. | */..) ws_ok=false ;;
            */[Ww][Ss]) ws_ok=false ;;
            */*) ;;
            *) ws_ok=false ;;
            esac
            ;;
        WS_HOOK_SECRET)
            WS_HOOK_SECRET=$ws_value
            case $ws_value in *[!A-Za-z0-9._-]*) ws_ok=false ;; esac
            [ "${#ws_value}" -le 64 ] || ws_ok=false
            ;;
        *) ws_die "$1, line $ws_n: unknown key $ws_key" ;;
        esac
        [ -z "$ws_value" ] || $ws_ok ||
            ws_die "$1, line $ws_n: $ws_key=$ws_value is not $(ws_conf_form "$ws_key")"
    done <"$1"
}

# ws_lan_host CIDR ADDRESS succeeds where ADDRESS is a host's address in the network CIDR: inside
# it, and neither its first address, the network's own, nor its last, its broadcast. In a /31 or a
# /32, every address is a host's.
ws_lan_host_old() {
    ws_cidr_contains "$1" "$2" || return 1
    ws_last=$(((1 << (32 - ${1#*/})) - 1))
    ws_host=$(($(ws_ipv4_int "$2") & ws_last))
    [ "$ws_last" -lt 3 ] || { [ "$ws_host" != 0 ] && [ "$ws_host" != "$ws_last" ]; }
}
