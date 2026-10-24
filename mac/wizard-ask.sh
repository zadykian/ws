# shellcheck shell=sh
# The wizard's Answers stage: the values that mac.conf lacks, each from the environment, under its
# key's name. It calls the functions of lib.sh and setup-lib.sh, and reads wizard.sh's lock and
# WS_CONF, mac.conf's path.

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
