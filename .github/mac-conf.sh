# shellcheck shell=sh
# The cases of the mac and mac-cld jobs write and read a conf, the file that WS_VALUES
# names, through these functions. A conf in /etc/ws is root's, and they write it through sudo. A
# conf is YAML, a KEY: value a line, and a case writes each value as the file holds it, in quotes
# where YAML needs them. They source this file and never run it. A conf's layout lives here alone:
# the Mac's values.yml holds the Mac's three keys, and servers/ws/values.yml, beside it, every other
# key. roles/client/values/scope.go lists the three keys too.

# conf_server names the server whose file goes with a conf. A case sets it to another name, and
# sets it back, where it works on a second server.
conf_server=ws

# conf_server_file CONF prints the path of the server's file that goes with CONF.
conf_server_file() {
    echo "${1%/*}/servers/$conf_server/values.yml"
}

# conf_split SCOPE prints the lines of stdin that go in the file of SCOPE, mac or server. A line
# with no key stays with values.yml.
conf_split() {
    awk -v scope="$1" '
        BEGIN {
            n = split("WS_GIT_SIGNING_PUB_KEY WS_IDENTITY_PUB_KEY WS_IDENTITY_AGENT", keys, " ")
            for (i = 1; i <= n; i++) mac[keys[i]] = 1
        }
        {
            key = $0
            server = 0
            if (key ~ /^[A-Za-z_][A-Za-z0-9_]*:/) {
                sub(/:.*/, "", key)
                server = !(key in mac)
            }
            if (server == (scope == "server")) print
        }'
}

# conf_as FILE CMD... runs CMD through sudo where FILE is in /etc/ws, root's, and as the runner
# elsewhere, as ws client does.
conf_as() {
    case $1 in
    /etc/ws/*)
        shift
        sudo "$@"
        ;;
    *)
        shift
        "$@"
        ;;
    esac
}

# conf_write CONF puts each KEY: value line of stdin in the file of its key, CONF or the server's
# file, in place of what they held. The server's file goes, with its directory, where it gets no
# line. A server's file that it makes and that holds no WS_SLOT gets WS_SLOT: 0 first, as values.yml
# holds it. WS_TUNNEL_PORT: 51820 follows, where the lines hold no WS_TUNNEL_PORT. It reads stdin first, so that
# conf_print CONF | ... | conf_write CONF is safe. It writes in place, through tee, so that CONF
# keeps its inode and its mode, and through sudo where CONF is in /etc/ws.
conf_write() {
    conf_as "$1" mkdir -p "${1%/*}"
    conf_lines=$(
        cat
        echo .
    )
    conf_lines=${conf_lines%.}
    printf '%s' "$conf_lines" | conf_split mac | conf_as "$1" tee "$1" >/dev/null
    conf_part=$(
        printf '%s' "$conf_lines" | conf_split server
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

# conf_print CONF prints the lines of CONF, then those of the server's file where there is one.
conf_print() {
    cat "$1"
    conf_part=$(conf_server_file "$1")
    [ ! -f "$conf_part" ] || cat "$conf_part"
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
