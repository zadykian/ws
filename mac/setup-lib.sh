# shellcheck shell=sh source-path=SCRIPTDIR
# The functions of lib.sh's that mac/setup.sh and mac/wizard.sh use, which each sources after
# lib.sh. They take setup.sh's arguments, check mac.conf's values, write the user's files and copy
# programs into root's. They call lib.sh's functions and read and set its variables, as those of
# lib.sh do.

# The directory of root's copies of ws-tunnel, its functions, wg and wireguard-go, which setup.sh
# makes and the tunnel's daemon runs.
# shellcheck disable=SC2034 # for setup.sh
ws_libexec=/usr/local/libexec/ws-tunnel

# ws_is_port PORT succeeds where PORT is a port number, 1 to 65535.
ws_is_port() {
    case $1 in
    '' | 0* | *[!0-9]* | ??????*) return 1 ;;
    esac
    [ "$1" -le 65535 ]
}

# ws_is_key KEY succeeds where KEY is a WireGuard public key: 43 base64 characters and an =.
ws_is_key() {
    case $1 in
    *[!A-Za-z0-9+/=]* | *=?*) return 1 ;;
    esac
    [ "${#1}" = 44 ] && [ "${1%=}" != "$1" ]
}

# ws_is_user NAME succeeds where NAME is a user name ssh can take: letters, digits, ., _ and -,
# with no - first, and 32 characters at most.
ws_is_user() {
    case $1 in
    '' | -* | *[!A-Za-z0-9._-]*) return 1 ;;
    esac
    [ "${#1}" -le 32 ]
}

# ws_are_networks LIST succeeds where LIST is IPv4 networks split by blanks, each as WS_LAN takes
# one. Its characters are checked first, so that the bare $1 globs nothing as it splits.
# shellcheck disable=SC2154 # ws_cable_net and ws_tunnel_net are lib.sh's
ws_are_networks() {
    case $1 in
    *[!0-9./[:blank:]]*) return 1 ;;
    esac
    for ws_net in $1; do
        ws_is_cidr "$ws_net" && ! ws_cidr_overlaps "$ws_net" "$ws_cable_net" &&
            ! ws_cidr_overlaps "$ws_net" "$ws_tunnel_net" || return 1
    done
}

# ws_are_resolvers LIST succeeds where LIST is DOMAIN=ADDRESS[,ADDRESS...] split by blanks, with up
# to 3 IPv4 addresses, as ws_resolver takes them. Each domain comes once and in lower case, as
# /etc/resolver's volume ignores case by default: two spellings would share one file.
ws_are_resolvers() {
    case $1 in
    *[!a-z0-9.,=[:blank:]-]*) return 1 ;;
    esac
    ws_domains=' '
    for ws_pair in $1; do
        case $ws_pair in *=*) ;; *) return 1 ;; esac
        ws_domain=${ws_pair%%=*}
        ws_is_domain "$ws_domain" || return 1
        case $ws_domains in *" $ws_domain "*) return 1 ;; esac
        ws_domains="$ws_domains$ws_domain "
        ws_addresses=${ws_pair#*=},
        ws_count=0
        while [ -n "$ws_addresses" ]; do
            ws_is_ipv4 "${ws_addresses%%,*}" || return 1
            ws_addresses=${ws_addresses#*,}
            ws_count=$((ws_count + 1))
        done
        [ "$ws_count" -le 3 ] || return 1
    done
}

# ws_args takes a script's arguments: --check sets WS_CHECK.
# shellcheck disable=SC2034,SC2154 # lib.sh's writers read WS_CHECK, and lib.sh sets ws_prog
ws_args() {
    for ws_arg; do
        case $ws_arg in
        --check) WS_CHECK=1 ;;
        -h | --help)
            printf 'usage: %s [--check]\n' "$ws_prog"
            exit 0
            ;;
        *) ws_die "unknown argument '$ws_arg'; usage: $ws_prog [--check]" ;;
        esac
    done
}

# The functions that read mac.conf are setup-conf.sh's, and those that set ws's key from it are
# setup-identity.sh's. Both are sourced from dir, mac/'s path, which setup.sh sets and any other
# script that sources setup-lib.sh must set too.
# shellcheck source=setup-conf.sh disable=SC2154 # dir is setup.sh's
. "$dir/setup-conf.sh"
# shellcheck source=setup-identity.sh disable=SC2154 # dir is setup.sh's
. "$dir/setup-identity.sh"

# ws_mode FILE prints FILE's permissions in three octal digits, as 600, with GNU's stat or the BSD
# stat of macOS, leaving out the setuid, setgid and sticky bits. It follows a link, as chmod does,
# since a link's own mode is 755 on macOS whatever its target's.
ws_mode() {
    ws_m=$(stat -L -c %a "$1" 2>/dev/null || stat -L -f %Lp "$1") || return 1
    ws_m=00$ws_m
    echo "${ws_m#"${ws_m%???}"}"
}

# ws_dir DIR MODE makes the user's directory DIR with MODE where it is missing, and sets MODE
# where it differs.
ws_dir() {
    if [ ! -d "$1" ]; then
        ws_changed=1
        if ws_checking; then
            ws_say "would make $1, mode $2"
        else
            mkdir -m "$2" "$1" || ws_die "cannot make $1"
        fi
    elif [ "$(ws_mode "$1")" != "$2" ]; then
        ws_changed=1
        if ws_checking; then
            ws_say "would set $1's mode to $2"
        else
            chmod "$2" "$1" || ws_die "cannot set $1's mode"
        fi
    fi
}

# ws_put FILE MODE [NAME] writes stdin to the user's FILE with MODE where its text or mode differs,
# through a new file renamed over it. NAME, FILE by default, labels WS_CHECK's diff.
# shellcheck disable=SC2154 # ws_tmp, lib.sh's, sets ws_tmpfile
ws_put() {
    ws_tmp
    ws_new=$ws_tmpfile
    cat >"$ws_new" || ws_die "cannot write $ws_new"
    if [ -f "$1" ] && cmp -s "$1" "$ws_new"; then
        rm -f "$ws_new"
        if [ "$(ws_mode "$1")" != "$2" ]; then
            ws_changed=1
            if ws_checking; then
                ws_say "would set $1's mode to $2"
            else
                chmod "$2" "$1" || ws_die "cannot set $1's mode"
            fi
        fi
        return 0
    fi
    ws_changed=1
    if ws_checking; then
        if [ -f "$1" ]; then
            ws_diff "$1" "$ws_new" "${3:-$1}"
        else
            ws_diff /dev/null "$ws_new" "${3:-$1}"
        fi
        rm -f "$ws_new"
        return 0
    fi
    ws_next="${1%/*}/.${1##*/}.ws-new"
    if cp "$ws_new" "$ws_next" && chmod "$2" "$ws_next" && mv -f "$ws_next" "$1"; then
        rm -f "$ws_new"
        ws_say "wrote $1"
    else
        rm -f "$ws_new" "$ws_next"
        ws_die "cannot write $1"
    fi
}

# ws_copy_root FROM FILE MODE copies FROM, a program, to FILE, root's, with MODE, where it differs.
# shellcheck disable=SC2034 # lib.sh and setup.sh read ws_changed
ws_copy_root() {
    ws_attrs=$(ws_root_attrs "$2")
    if [ -n "$ws_attrs" ] && sudo cmp -s "$1" "$2"; then
        ws_root_fix "$2" "$3" "$ws_attrs"
        return 0
    fi
    ws_changed=1
    if ws_checking; then
        ws_say "would copy $1 to $2"
    elif sudo sh -c 'cp "$1" "$2" && chown 0:0 "$2" && chmod "$3" "$2" && mv -f "$2" "$4"' \
        sh "$1" "${2%/*}/.${2##*/}.ws-new" "$3" "$2"; then
        ws_say "copied $1 to $2"
    else
        sudo rm -f "${2%/*}/.${2##*/}.ws-new"
        ws_die "cannot copy $1 to $2"
    fi
}
