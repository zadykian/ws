# shellcheck shell=sh
# The cases of the client and client-cld jobs write and read a conf, the directory that WS_VALUES
# names, through these functions. A conf in /etc/ws is root's, and they write it through sudo. A
# server's file is YAML, a KEY: value a line, and a case writes each value as the file holds it, in
# quotes where YAML needs them. They source this file and never run it. A conf's layout lives here
# alone: each server's keys are in CONF/servers/NAME/values.yml.

# conf_server names the server whose file a conf holds for the functions. A case sets it to another
# name, and sets it back, where it works on a second server.
conf_server=ws

# conf_server_file CONF prints the path of the server's file in CONF.
conf_server_file() {
    echo "$1/servers/$conf_server/values.yml"
}

# conf_as CONF CMD... runs CMD through sudo where CONF is in /etc/ws, root's, and as the runner
# elsewhere, as ws macos does.
conf_as() {
    case $1 in
    /etc/ws | /etc/ws/*)
        shift
        sudo "$@"
        ;;
    *)
        shift
        "$@"
        ;;
    esac
}

# conf_write CONF puts the lines of stdin, KEY: value each, in the server's file of CONF, in place of
# what it held. The file goes, with its directory, where it gets no line. A file that it makes and
# that holds no WS_SLOT gets WS_SLOT: 0 first. WS_TUNNEL_PORT: 51820 follows, where the lines hold
# no WS_TUNNEL_PORT. It reads stdin first, so that conf_print CONF | ... | conf_write CONF is safe.
# It writes in place, through tee, so that the file keeps its inode and its mode, and through sudo
# where CONF is in /etc/ws.
conf_write() {
    conf_part=$(
        cat
        echo .
    )
    conf_part=${conf_part%.}
    conf_dest=$(conf_server_file "$1")
    if [ -n "$conf_part" ]; then
        conf_as "$1" mkdir -p "${conf_dest%/*}"
        if ! printf '%s' "$conf_part" | grep -q '^WS_SLOT:'; then
            printf '%s' "$conf_part" | grep -q '^WS_TUNNEL_PORT:' || conf_part="WS_TUNNEL_PORT: 51820
$conf_part"
            conf_part="WS_SLOT: 0
$conf_part"
        fi
        printf '%s' "$conf_part" | conf_as "$1" tee "$conf_dest" >/dev/null
    else
        conf_as "$1" rm -rf "${conf_dest%/*}"
        conf_as "$1" rmdir "${conf_dest%/*/*}" 2>/dev/null || true
    fi
}

# conf_print CONF prints the lines of the server's file in CONF.
conf_print() {
    cat "$(conf_server_file "$1")"
}

# conf_set CONF LINE... handles each LINE in turn. A KEY: value LINE takes the place of KEY's first
# line and drops the later ones, or goes at the end where CONF has no KEY. A bare KEY drops its
# lines.
conf_set() {
    conf_file=$1
    shift
    for conf_arg in "$@"; do
        conf_print "$conf_file" | conf_key=${conf_arg%%:*} conf_line=$conf_arg awk '
            BEGIN {
                key = ENVIRON["conf_key"]
                line = ENVIRON["conf_line"]
                n = length(key) + 1
            }
            substr($0, 1, n) == key ":" {
                if (!done && line != key) print line
                done = 1
                next
            }
            { print }
            END { if (!done && line != key) print line }
        ' | conf_write "$conf_file"
    done
}
