# shellcheck shell=sh
# The wizard's Answers stage: the values that mac.conf lacks, each from the environment, under its
# key's name, or asked at the terminal. wizard.sh sources it before any stage reads mac.conf, and
# runs the stage before setup.sh's. It calls the functions of lib.sh and setup-lib.sh, and reads
# wizard.sh's lock and WS_CONF, mac.conf's path.

# The environment's answers, kept before ws_conf sets each key's variable: ask_set_KEY is 1 where
# the environment sets KEY, even to nothing, and ask_env_KEY holds its value.
for ask_key in WS_TARGET WS_LAN WS_LAN_ADDRESS WS_REMOTE WS_REMOTE_PORT WS_GIT_NAME WS_GIT_EMAIL \
    WS_TUNNEL_NETWORKS WS_RESOLVERS WS_HOOK WS_HOOK_SECRET WS_CABLE_SERVICE WS_CABLE_PORT; do
    eval "ask_set_$ask_key=\${$ask_key+1} ask_env_$ask_key=\${$ask_key-}"
done

# ask_keys asks the stage's keys in order, each with its question and default. The machine that ws
# runs on comes first. ws's address and git's email are asked only with a LAN and a name, and need a
# value then; the router's port only with its name, and the Keychain item only with another
# repository. The address defaults to the router's lease where the LAN holds it (docs/router.md),
# and git's identity to ws's own.
ask_keys() {
    ask WS_TARGET "The machine ws runs on: lan, cloud or mac-vm" lan \
        "WS_TARGET needs lan, cloud or mac-vm"
    # A VM on the Mac needs a LAN, and a cloud VM a name and a port.
    ask_need=
    [ "$WS_TARGET" != mac-vm ] || ask_need="WS_TARGET=mac-vm needs WS_LAN"
    ask WS_LAN "ws's home LAN" 10.88.0.0/24 "$ask_need"
    if [ -n "$WS_LAN" ]; then
        ask_default=
        ! ws_lan_host "$WS_LAN" 10.88.0.10 || ask_default=10.88.0.10
        ask WS_LAN_ADDRESS "ws's address on the LAN" "$ask_default" \
            "WS_LAN and WS_LAN_ADDRESS go together"
    fi
    ask_need=
    [ "$WS_TARGET" != cloud ] || ask_need="WS_TARGET=cloud needs WS_REMOTE and WS_REMOTE_PORT"
    ask WS_REMOTE "The router's name on the internet" '' "$ask_need"
    [ -z "$WS_REMOTE" ] ||
        ask WS_REMOTE_PORT "The router's port forwarded to ws's 22" 22022 "$ask_need"
    ask WS_GIT_NAME "git's name on the Mac" "$(ask_role shell_git_name)"
    if [ -n "$WS_GIT_NAME" ]; then
        ask WS_GIT_EMAIL "git's email on the Mac" "$(ask_role shell_git_email)" \
            "WS_GIT_NAME and WS_GIT_EMAIL go together"
    fi
    ask WS_TUNNEL_NETWORKS "More networks through the tunnel, split by blanks" ''
    ask WS_RESOLVERS "Nameservers behind the tunnel, as DOMAIN=ADDRESS, split by blanks" ''
    ask WS_HOOK "Another repository whose apply.sh runs on ws, as OWNER/REPO" ''
    [ -z "$WS_HOOK" ] || ask WS_HOOK_SECRET "The Keychain item of its apply.sh's secret" ''
}

# ask_role NAME prints the shell role's default NAME, which ws's git takes.
# shellcheck disable=SC2154 # dir is wizard.sh's
ask_role() {
    sed -n "s/^$1: *//p" "$dir/../roles/shell/defaults/main.yml" 2>/dev/null
}

# ask KEY QUESTION DEFAULT [NEED] gives KEY a value where mac.conf has no line for it: the
# environment's, else the answer to QUESTION at the terminal, stdin. A bad answer is asked again,
# and the environment's stops the run. With NEED, ws_conf's message for KEY's pair, no value is
# bad. It sets KEY and adds its line to ask_new. For ask_done, under ask_reading, it only exits
# where the line is missing.
ask() {
    ask_lacks "$1" || return 0
    [ -z "${ask_reading-}" ] || exit 1
    eval "ask_set=\${ask_set_$1-} ask_value=\${ask_env_$1-}"
    if [ -n "$ask_set" ]; then
        ask_check "$1" "$ask_value" "${4-}" ||
            ws_die "Answers: $1 comes from the environment: set it to such a value, or unset it"
    else
        [ -t 0 ] || ws_die "Answers: no terminal to ask $1: set $1"
        until ask_read "$1" "$2" "$3" "${4-}" && ask_check "$1" "$ask_value" "${4-}"; do :; done
    fi
    eval "$1=\$ask_value"
    ask_new="$ask_new$(ask_line "$1" "$ask_value")
"
}

# ask_lacks KEY succeeds where mac.conf has no line for KEY, or is missing.
ask_lacks() {
    ! grep -q "^[[:blank:]]*$1=" "$WS_CONF" 2>/dev/null
}

# ask_read KEY QUESTION DEFAULT NEED prints QUESTION, with DEFAULT and none in brackets where they
# serve. It sets ask_value to the line read: DEFAULT for an empty one, nothing for none, and a pair
# of quotes taken off, as ws_conf does. The first says how to answer.
ask_read() {
    if [ -z "${ask_told-}" ]; then
        ws_say "Answers: values for $WS_CONF; an empty line takes the one in brackets," \
            "and none leaves it empty"
        ask_told=1
    fi
    ask_hint=$3
    [ -n "$4" ] || ask_hint=${3:+$3, or }none
    printf '%s (%s)%s: ' "$2" "$1" "${ask_hint:+ [$ask_hint]}" >&2
    read -r ask_value || ws_die "Answers: the terminal ended before $1's answer: set $1"
    case $ask_value in
    '') ask_value=$3 ;;
    none) ask_value= ;;
    \"*\" | \'*\')
        ask_value=${ask_value#?}
        ask_value=${ask_value%?}
        ;;
    esac
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

# ask_write LINES [STAGE] writes mac.conf with LINES, through ws_put, where the whole passes
# ws_conf: a line takes the place of mac.conf's lines for its key, else goes at the end. A new file
# gets mode 644, in a directory it makes. ws_conf's messages name mac.conf, and its own name STAGE,
# Answers by default. Where mac.conf is a link, as a dotfiles manager makes it, it prints LINES and
# stops. Under --check, it prints the diff, under mac.conf's own path, and points WS_CONF at the
# file it would write, in the lock, for setup.sh --check and the stages after.
# shellcheck disable=SC2154 # lock is wizard.sh's
ask_write() {
    ask_file=$lock/mac.conf.new
    ask_stage=${2:-Answers}
    ask_conf=${ask_conf:-$WS_CONF}
    {
        if [ -s "$WS_CONF" ]; then
            ask_lines=$1 awk '
                function key(l) { sub(/^[ \t]+/, "", l); sub(/=.*/, "", l); return l }
                BEGIN {
                    n = split(ENVIRON["ask_lines"], line, "\n")
                    for (i = 1; i <= n; i++) if ((k[i] = key(line[i])) != "") at[k[i]] = i
                }
                { this = key($0) }
                this in at { if (!(this in done)) print line[at[this]]; done[this] = 1; next }
                { print }
                END { for (i = 1; i <= n; i++) if (k[i] != "" && !(k[i] in done)) print line[i] }
            ' "$WS_CONF" || ws_die "$ask_stage: cannot read $WS_CONF"
        else
            printf '%s' "$1"
        fi
    } >"$ask_file" || ws_die "$ask_stage: cannot write $ask_file"
    (ws_conf_lines "$ask_file" && ws_conf_pairs "$ask_conf") || return 1
    if [ -L "$WS_CONF" ]; then
        ws_say "$ask_stage: $WS_CONF is a link to $(readlink "$WS_CONF"), which the wizard" \
            "leaves: add these lines to its file, then run the wizard again"
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
        mkdir -p "$ask_dir" || ws_die "$ask_stage: cannot make $ask_dir"
    fi
    ws_put "$WS_CONF" "$ask_mode" "$ask_conf" <"$ask_file"
    if ws_checking; then
        mv -f "$ask_file" "$lock/mac.conf" && WS_CONF=$lock/mac.conf
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
