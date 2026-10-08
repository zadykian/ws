# shellcheck shell=sh
# The functions of ws's scripts for the Mac, sourced by mac/setup.sh. Sourcing it defines
# functions and variables whose names start with ws_, and changes nothing else.
#
# Every function that writes reads WS_CHECK: set to anything but empty or 0, as ws_args sets it
# for --check, it prints each change as a diff, or a line where there is no text to compare, and
# makes none. A function exits with status 1 on an error, with a message on stderr.

ws_prog=${0##*/}
# The subnets of the cable and of the tunnel, which the home LAN must stay clear of.
ws_cable_net=10.77.0.0/30
ws_tunnel_net=10.99.0.0/24

# ws_say prints a message of the script's on stderr.
ws_say() {
    printf '%s: %s\n' "$ws_prog" "$*" >&2
}

# ws_die prints a message of the script's and exits with status 1.
ws_die() {
    ws_say "$*"
    exit 1
}

# ws_checking succeeds under WS_CHECK.
ws_checking() {
    case ${WS_CHECK-} in
    '' | 0) return 1 ;;
    esac
}

# ws_args takes a script's arguments: --check sets WS_CHECK.
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

# ws_is_ipv4 ADDRESS succeeds where ADDRESS is an IPv4 address in four decimal parts, none with a
# leading zero, which the shell's arithmetic would read as octal.
ws_is_ipv4() {
    case $1 in
    *[!0-9.]* | *..* | .* | *.) return 1 ;;
    esac
    ws_rest=$1.
    ws_parts=0
    while [ -n "$ws_rest" ]; do
        ws_part=${ws_rest%%.*}
        ws_rest=${ws_rest#*.}
        ws_parts=$((ws_parts + 1))
        case $ws_part in
        [0-9] | [1-9][0-9] | 1[0-9][0-9] | 2[0-4][0-9] | 25[0-5]) ;;
        *) return 1 ;;
        esac
    done
    [ "$ws_parts" = 4 ]
}

# ws_ipv4_int ADDRESS prints a valid IPv4 address as a number.
ws_ipv4_int() {
    ws_rest=$1.
    ws_int=0
    while [ -n "$ws_rest" ]; do
        ws_int=$((ws_int * 256 + ${ws_rest%%.*}))
        ws_rest=${ws_rest#*.}
    done
    echo "$ws_int"
}

# ws_is_cidr CIDR succeeds where CIDR is an IPv4 network, ADDRESS/LENGTH, with no bits set past
# its length: 10.88.0.0/24, not 10.88.0.10/24.
ws_is_cidr() {
    case $1 in
    */[0-9] | */[12][0-9] | */3[0-2]) ;;
    *) return 1 ;;
    esac
    ws_is_ipv4 "${1%/*}" || return 1
    ws_int=$(ws_ipv4_int "${1%/*}")
    [ $((ws_int & ((1 << (32 - ${1#*/})) - 1))) = 0 ]
}

# ws_cidr_contains CIDR ADDRESS succeeds where the network CIDR holds ADDRESS.
ws_cidr_contains() {
    ws_shift=$((32 - ${1#*/}))
    [ $(($(ws_ipv4_int "${1%/*}") >> ws_shift)) = $(($(ws_ipv4_int "$2") >> ws_shift)) ]
}

# ws_cidr_overlaps CIDR CIDR succeeds where the two networks share an address: one holds the
# other's first address. 0.0.0.0/0 overlaps every network.
ws_cidr_overlaps() {
    ws_cidr_contains "$1" "${2%/*}" || ws_cidr_contains "$2" "${1%/*}"
}

# ws_is_port PORT succeeds where PORT is a port number, 1 to 65535.
ws_is_port() {
    case $1 in
    '' | 0* | *[!0-9]* | ??????*) return 1 ;;
    esac
    [ "$1" -le 65535 ]
}

# ws_is_domain NAME succeeds where NAME is one or more DNS labels joined by dots: each of 1 to 63
# letters, digits and hyphens, with no hyphen first or last, and 253 characters in all.
ws_is_domain() {
    [ -n "$1" ] && [ "${#1}" -le 253 ] || return 1
    ws_rest=$1.
    while [ -n "$ws_rest" ]; do
        ws_part=${ws_rest%%.*}
        ws_rest=${ws_rest#*.}
        case $ws_part in
        '' | -* | *- | *[!A-Za-z0-9-]*) return 1 ;;
        esac
        [ "${#ws_part}" -le 63 ] || return 1
    done
}

# ws_conf FILE reads the Mac's settings from FILE, ~/.config/ws/mac.conf, without running it: a
# KEY=value a line, the value bare or in a pair of quotes, blank lines and lines starting with #
# skipped. It stops on any other line, any other key, and a value of the wrong form.
ws_conf() {
    [ -f "$1" ] || ws_die "$1 is missing: README.md's The Mac says what goes in it"
    WS_LAN=
    WS_LAN_ADDRESS=
    WS_REMOTE=
    WS_REMOTE_PORT=
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
        *) ws_die "$1, line $ws_n: unknown key $ws_key" ;;
        esac
        $ws_ok || ws_die "$1, line $ws_n: $ws_key=$ws_value is not $(ws_conf_form "$ws_key")"
    done <"$1"
    case ${WS_LAN:+lan}${WS_LAN_ADDRESS:+address} in
    lan | address) ws_die "$1: WS_LAN and WS_LAN_ADDRESS go together" ;;
    esac
    if [ -n "$WS_REMOTE_PORT" ] && [ -z "$WS_REMOTE" ]; then
        ws_die "$1: WS_REMOTE_PORT needs WS_REMOTE"
    fi
}

# ws_conf_form KEY prints the form a value of KEY takes, for ws_conf's message.
ws_conf_form() {
    case $1 in
    WS_LAN) echo "an IPv4 network, as 10.88.0.0/24, clear of $ws_cable_net and $ws_tunnel_net" ;;
    WS_LAN_ADDRESS) echo "an IPv4 address" ;;
    WS_REMOTE) echo "a host name or an IPv4 address" ;;
    *) echo "a port, 1 to 65535" ;;
    esac
}

# ws_mode FILE prints FILE's permissions in octal, as 600, with GNU's stat or the BSD stat of
# macOS.
ws_mode() {
    stat -c %a "$1" 2>/dev/null || stat -f %Lp "$1"
}

# ws_tmp makes an empty file of the user's own for a function's text, and sets ws_tmpfile to its
# name.
ws_tmp() {
    ws_tmpfile=$(mktemp "${TMPDIR:-/tmp}/ws.XXXXXXXX") || ws_die "cannot make a temporary file"
}

# ws_diff OLD NEW NAME prints the diff from OLD to NEW, each labelled NAME, or /dev/null for a
# file that is missing.
ws_diff() {
    ws_from=$3
    [ "$1" != /dev/null ] || ws_from=/dev/null
    ws_to=$3
    [ "$2" != /dev/null ] || ws_to=/dev/null
    diff -u -L "$ws_from" -L "$ws_to" "$1" "$2" || true
}

# ws_dir DIR MODE makes the user's directory DIR with MODE where it is missing, and sets MODE
# where it differs.
ws_dir() {
    if [ ! -d "$1" ]; then
        if ws_checking; then
            ws_say "would make $1, mode $2"
        else
            mkdir -m "$2" "$1" || ws_die "cannot make $1"
        fi
    elif [ "$(ws_mode "$1")" != "$2" ]; then
        if ws_checking; then
            ws_say "would set $1's mode to $2"
        else
            chmod "$2" "$1" || ws_die "cannot set $1's mode"
        fi
    fi
}

# ws_put FILE MODE writes stdin to the user's FILE with MODE where its text or mode differs,
# through a new file renamed over it.
ws_put() {
    ws_tmp
    ws_new=$ws_tmpfile
    cat >"$ws_new" || ws_die "cannot write $ws_new"
    if [ -f "$1" ] && cmp -s "$1" "$ws_new"; then
        rm -f "$ws_new"
        if [ "$(ws_mode "$1")" != "$2" ]; then
            if ws_checking; then
                ws_say "would set $1's mode to $2"
            else
                chmod "$2" "$1" || ws_die "cannot set $1's mode"
            fi
        fi
        return 0
    fi
    if ws_checking; then
        if [ -f "$1" ]; then
            ws_diff "$1" "$ws_new" "$1"
        else
            ws_diff /dev/null "$ws_new" "$1"
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

# ws_include_first FILE LINE puts LINE, an Include, above every setting of the ssh_config FILE,
# as ssh keeps the first value it reads for a key. It makes FILE where it is missing, and adds
# LINE at the top where FILE lacks it, keeping the old file as FILE.TIME~. It stops where LINE
# sits below a setting, and where FILE is a link, which it would replace with a file.
ws_include_first() {
    [ ! -L "$1" ] || ws_die "$1 is a link: put '$2' at the top of the file it points to"
    if [ ! -e "$1" ]; then
        ws_put "$1" 600 <<EOF
$2
EOF
        return 0
    fi
    ws_at=$(awk -v want="$2" '
        {
            line = $0
            sub(/^[ \t]+/, "", line)
            sub(/[ \t\r]+$/, "", line)
            gsub(/[ \t]+/, " ", line)
        }
        line == "" || line ~ /^#/ { next }
        tolower(line) == tolower(want) { print (++settings == 1 ? "first" : NR); exit }
        { settings++ }
    ' "$1") || ws_die "cannot read $1"
    case $ws_at in
    first) return 0 ;;
    '') ;;
    *) ws_die "line $ws_at of $1 is '$2', below other settings, which ssh would take first: move it to the top" ;;
    esac
    if ! ws_checking; then
        ws_backup=$1.$(date +%Y%m%d%H%M%S)~
        cp -p "$1" "$ws_backup" || ws_die "cannot keep $1 as $ws_backup"
        ws_say "kept the old $1 as $ws_backup"
    fi
    ws_tmp
    ws_top=$ws_tmpfile
    { printf '%s\n\n' "$2" && cat "$1"; } >"$ws_top" || ws_die "cannot write $ws_top"
    ws_put "$1" "$(ws_mode "$1")" <"$ws_top"
    rm -f "$ws_top"
}

# ws_own_hosts FILE NAME... warns of each line of the ssh_config FILE that starts a Host or Match
# block for one of the NAMEs, whose settings come after those config.d's first line gives.
ws_own_hosts() {
    [ -f "$1" ] || return 0
    ws_file=$1
    shift
    awk -v names=" $* " '
        {
            key = tolower($1)
            if (key != "host" && key != "match") next
            for (i = 2; i <= NF; i++) {
                if (index(names, " " $i " ")) { print NR ": " $0; next }
            }
        }
    ' "$ws_file" | while IFS= read -r ws_hit; do
        ws_say "warning: line ${ws_hit%%:*} of $ws_file,${ws_hit#*:}, sets its own values for ws: those in config.d, read first, win where both set a key"
    done
}
