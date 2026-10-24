# shellcheck shell=sh
# The wizard's Answers stage: the values that mac.conf lacks, each from the environment, under its
# key's name. wizard.sh sources it before any stage reads mac.conf, and runs the stage before
# setup.sh's. It calls the functions of lib.sh and setup-lib.sh, and reads wizard.sh's lock and
# WS_CONF, mac.conf's path.

# The environment's answers, kept before ws_conf sets each key's variable: ask_set_KEY is 1 where
# the environment sets KEY, even to nothing, and ask_env_KEY holds its value.
for ask_key in WS_LAN WS_LAN_ADDRESS WS_REMOTE WS_REMOTE_PORT WS_GIT_NAME WS_GIT_EMAIL \
    WS_TUNNEL_NETWORKS WS_RESOLVERS; do
    eval "ask_set_$ask_key=\${$ask_key+1} ask_env_$ask_key=\${$ask_key-}"
done

# ask_keys gives the stage's keys their values, in order. ws's address and git's email come only
# with a LAN and a name, and need a value then; the router's port only with its name.
ask_keys() {
    ask WS_LAN
    if [ -n "$WS_LAN" ]; then
        ask WS_LAN_ADDRESS "WS_LAN and WS_LAN_ADDRESS go together"
    fi
    ask WS_REMOTE
    [ -z "$WS_REMOTE" ] || ask WS_REMOTE_PORT
    ask WS_GIT_NAME
    if [ -n "$WS_GIT_NAME" ]; then
        ask WS_GIT_EMAIL "WS_GIT_NAME and WS_GIT_EMAIL go together"
    fi
    ask WS_TUNNEL_NETWORKS
    ask WS_RESOLVERS
}

# ask KEY [NEED] gives KEY the environment's value where mac.conf has no line for it, and stops the
# run where the environment has none, or a bad one. With NEED, ws_conf's message for KEY's pair, no
# value is bad. It sets KEY and adds its line to ask_new. For ask_done, under ask_reading, it only
# exits where the line is missing.
# shellcheck disable=SC2154 # the eval sets ask_set and ask_value
ask() {
    ask_lacks "$1" || return 0
    [ -z "${ask_reading-}" ] || exit 1
    eval "ask_set=\${ask_set_$1-} ask_value=\${ask_env_$1-}"
    [ -n "$ask_set" ] || ws_die "Answers: no value for $1 in $WS_CONF or the environment: set $1"
    ask_check "$1" "$ask_value" "${2-}" ||
        ws_die "Answers: $1 comes from the environment: set it to such a value, or unset it"
    eval "$1=\$ask_value"
    ask_new="$ask_new$(ask_line "$1" "$ask_value")
"
}

# ask_lacks KEY succeeds where mac.conf has no line for KEY, or is missing.
ask_lacks() {
    ! grep -q "^[[:blank:]]*$1=" "$WS_CONF" 2>/dev/null
}

# ask_check KEY VALUE [NEED] succeeds where mac.conf can take VALUE for KEY, and otherwise says why:
# NEED where VALUE is empty, ws_conf's check of the line, and WS_LAN_ADDRESS inside WS_LAN. A
# newline in VALUE fails that check, as it would add a line of its own.
ask_check() {
    if [ -z "$2" ]; then
        [ -n "${3-}" ] || return 0
        ws_say "Answers: $3"
        return 1
    fi
    if [ "$(ask_line "$1" "$2" | wc -l)" -ne 1 ] ||
        ! ask_line "$1" "$2" | (ws_conf_lines /dev/stdin) 2>/dev/null; then
        ws_say "Answers: $1=$2 is not $(ws_conf_form "$1")"
        return 1
    fi
    if [ "$1" = WS_LAN_ADDRESS ] && ! ws_lan_host "$WS_LAN" "$2"; then
        ws_say "Answers: $1=$2 is not $(ws_conf_form "$1"), $WS_LAN"
        return 1
    fi
}

# ask_line KEY VALUE prints KEY's line for mac.conf, VALUE in quotes where it holds a blank.
ask_line() {
    case $2 in
    *[[:blank:]]*) printf '%s="%s"\n' "$1" "$2" ;;
    *) printf '%s=%s\n' "$1" "$2" ;;
    esac
}

# ask_write LINES writes mac.conf with LINES at its end, through ws_put, where the whole passes
# ws_conf: mode 644 for a new file, in a directory it makes. ws_conf's messages name mac.conf. Where
# mac.conf is a link, as a dotfiles manager makes it, it prints LINES and stops. Under --check, it
# prints the diff, and points WS_CONF at the file it would write, in the lock, for setup.sh --check.
# shellcheck disable=SC2154 # lock is wizard.sh's
ask_write() {
    ask_file=$lock/mac.conf
    {
        if [ -s "$WS_CONF" ]; then
            cat "$WS_CONF" || ws_die "Answers: cannot read $WS_CONF"
            [ -z "$(tail -c 1 "$WS_CONF")" ] || echo
        fi
        printf '%s' "$1"
    } >"$ask_file" || ws_die "Answers: cannot write $ask_file"
    (ws_conf_lines "$ask_file" && ws_conf_pairs "$WS_CONF") || return 1
    if [ -L "$WS_CONF" ]; then
        ws_say "Answers: $WS_CONF is a link to $(readlink "$WS_CONF"), which the wizard leaves:" \
            "add these lines to its file, then run the wizard again"
        printf '%s' "$1" >&2
        exit 1
    fi
    ask_mode=644
    [ ! -f "$WS_CONF" ] || ask_mode=$(ws_mode "$WS_CONF")
    ask_dir=$(dirname "$WS_CONF")
    if [ -d "$ask_dir" ]; then
        :
    elif ws_checking; then
        ws_say "would make $ask_dir"
    else
        mkdir -p "$ask_dir" || ws_die "Answers: cannot make $ask_dir"
    fi
    ws_put "$WS_CONF" "$ask_mode" <"$ask_file"
    if ws_checking; then
        WS_CONF=$ask_file
    fi
}

# ask_done succeeds where mac.conf passes ws_conf and has a line for each key that ask_keys asks.
ask_done() {
    [ -f "$WS_CONF" ] || return 1
    (
        ws_conf "$WS_CONF" 2>/dev/null
        ask_reading=1
        ask_keys
    )
}

# ask_do reads mac.conf's lines, and stops on a bad one, as setup.sh would, or where it can't read
# them. Then it asks for the keys that mac.conf lacks, and writes them.
ask_do() {
    ask_new=
    if [ -f "$WS_CONF" ]; then
        ws_conf_lines "$WS_CONF" ||
            ws_die "Answers: cannot read $WS_CONF; make it readable, then run the wizard again"
    else
        ws_conf_lines /dev/null
    fi
    ask_keys
    ask_write "$ask_new"
}

# ask_fail says what to do where mac.conf's keys don't go together, the answers among them.
ask_fail() {
    echo "setup.sh would refuse $WS_CONF${ask_new:+ with the answers}, as above; fix it, then run" \
        "the wizard again"
}
