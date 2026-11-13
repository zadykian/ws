# shellcheck shell=sh
# The cases of the mac, mac-wizard and mac-cld jobs write and read a conf, the file that WS_CONF
# names, through these functions. They source this file and never run it. A conf's layout lives
# here alone.

# conf_write CONF puts the KEY=value lines of stdin in CONF, in place of what it held. It reads all
# of stdin first, so that conf_print CONF | ... | conf_write CONF is safe, and it writes in place,
# so that CONF keeps its inode and its mode.
conf_write() {
    mkdir -p "${1%/*}"
    conf_lines=$(
        cat
        echo .
    )
    conf_lines=${conf_lines%.}
    printf '%s' "$conf_lines" >"$1"
}

# conf_print CONF prints the lines of CONF.
conf_print() {
    cat "$1"
}

# conf_set CONF LINE... handles each LINE in turn. A KEY=value LINE takes the place of KEY's first
# line and drops the later ones, or goes at the end where CONF has no KEY. A bare KEY drops its
# lines.
conf_set() {
    conf_file=$1
    shift
    for conf_arg in "$@"; do
        conf_print "$conf_file" | conf_key=${conf_arg%%=*} conf_line=$conf_arg awk '
            BEGIN {
                key = ENVIRON["conf_key"]
                line = ENVIRON["conf_line"]
                n = length(key) + 1
            }
            substr($0, 1, n) == key "=" {
                if (!done && line != key) print line
                done = 1
                next
            }
            { print }
            END { if (!done && line != key) print line }
        ' | conf_write "$conf_file"
    done
}
