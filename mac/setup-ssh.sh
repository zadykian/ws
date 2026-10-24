# shellcheck shell=sh
# The half of mac/setup.sh that writes ~/.ssh/config.d/ws and puts its Include first in
# ~/.ssh/config, with the functions that read and edit an ssh_config file. setup.sh sources it
# after lib.sh and setup-lib.sh, whose functions and variables it uses.

# ws's end of the cable, and its address in the tunnel.
cable_server=10.77.0.1
tunnel_server=10.99.0.1
# The start of config.d/ws's first line, by which a run knows the file for the script's own.
header="# Written by ws's mac/setup.sh"

# ssh_config prints ~/.ssh/config.d/ws from mac.conf's values and the IgnoreUnknown patterns $1.
# ssh keeps the first value it reads for a key, so each Match sets Port 22 before Host ws-ssh sets
# the forward's port, and the cable comes before the LAN. localnetwork matches the Mac's own
# addresses, with no probe. originalhost matches the name typed, which a HostName set above does
# not change, unlike Match host. Until ws goes through the tunnel, via_tunnel=1, ws takes the ways
# of ws-ssh too, so that an ssh ws that worked before goes on working.
# shellcheck disable=SC2154 # ws_cable_net is lib.sh's, and via_tunnel setup.sh's
ssh_config() {
    # The hosts of the direct ways, for Host and for Match originalhost's list.
    direct=ws-ssh
    listed=ws-ssh
    if [ "$via_tunnel" = 0 ]; then
        direct='ws ws-ssh'
        listed=ws,ws-ssh
    fi
    cat <<EOF
$header from ~/.config/ws/mac.conf: its next run replaces changes made here.

# Only Apple's ssh knows UseKeychain. ssh keeps the first IgnoreUnknown it reads, this one, so it
# carries the patterns of ~/.ssh/config's own too: run setup.sh again after a change to those.
IgnoreUnknown $1

# ws over the cable, where the Mac has an address on it.
Match originalhost $listed localnetwork $ws_cable_net
    HostName $cable_server
    Port 22
EOF
    if [ -n "$WS_LAN" ]; then
        cat <<EOF
# ws on the home LAN, where the Mac has an address there.
Match originalhost $listed localnetwork $WS_LAN
    HostName $WS_LAN_ADDRESS
    Port 22
EOF
    fi
    if [ -n "$WS_REMOTE_PORT" ]; then
        cat <<EOF
# Elsewhere, through the router's port forward, which takes IPv4 alone, while the router's name
# may have an AAAA record too.
Host $direct
    HostName $WS_REMOTE
    Port $WS_REMOTE_PORT
    AddressFamily inet
EOF
    fi
    if [ "$via_tunnel" = 1 ]; then
        cat <<EOF

# ws through the tunnel, at the one address it has there whichever way the Mac is connected, on
# ws's own port, whatever a Host * of the user's sets.
Host ws
    HostName $tunnel_server
    Port 22
EOF
    fi
    cat <<EOF

# One known_hosts entry, ws's, whichever way ssh goes. 30 s between checks, 3 of them unanswered
# before ssh gives up.
Host ws ws-ssh
    User $WS_USER
    HostKeyAlias ws
$(ws_identity_lines)
    ServerAliveInterval 30
EOF
}

# ssh_whole stops where ssh refuses the new config.d/ws, $1, read first, then ~/.ssh/config, then
# the system's file, as ssh will read them after the run. ~/.ssh/config's Include of config.d/ws,
# at line $2 where there is one, is left out, as it would read the file the run replaces.
# shellcheck disable=SC2154 # ws_tmp, lib.sh's, sets ws_tmpfile
ssh_whole() {
    user=$HOME/.ssh/config
    read_user=$user
    ws_tmp
    whole=$ws_tmpfile
    ws_tmp
    copy=$ws_tmpfile
    if [ -n "$2" ] && [ -f "$user" ]; then
        awk -v skip="$2" '{ print (NR == skip ? "" : $0) }' "$user" >"$copy" ||
            ws_die "cannot read $user"
        read_user=$copy
    fi
    # The system's file goes through an Include too, so ssh resolves its relative Includes under
    # ~/.ssh, and checks its mode, where ssh itself would not.
    {
        printf 'Include "%s"\n' "$1"
        [ ! -f "$user" ] || printf 'Match all\nInclude "%s"\n' "$read_user"
        [ ! -f /etc/ssh/ssh_config ] || printf 'Match all\nInclude /etc/ssh/ssh_config\n'
    } >"$whole" || ws_die "cannot write $whole"
    if ! err=$(ssh -G -F "$whole" ws 2>&1 >/dev/null) ||
        ! err=$(ssh -G -F "$whole" ws-ssh 2>&1 >/dev/null); then
        rm -f "$whole" "$copy" "$1"
        err=$(printf '%s\n' "$err" | sed "s|$copy|$user|g")
        case $err in
        *"$HOME/.ssh/config.d/ws"*)
            ws_die "ssh refuses the old config.d/ws, which ~/.ssh/config also reads, as $err:" \
                "move it away, and run setup.sh again"
            ;;
        *'Bad configuration option'*)
            ws_die "ssh refuses ~/.ssh/config read after the new config.d/ws, as $err;" \
                "config.d/ws's IgnoreUnknown, read first, carries only ~/.ssh/config's own" \
                "patterns, so one for a file it includes must be there too"
            ;;
        esac
        ws_die "ssh refuses ~/.ssh/config read after the new config.d/ws, as $err"
    fi
    rm -f "$whole" "$copy"
}

# ssh_files writes ~/.ssh/config.d/ws and puts its Include first in ~/.ssh/config. Before it
# writes, it stops where config.d/ws is a file it did not write, and where the Include could not go
# first. It stops or warns where ws_identity does, and stops where ssh refuses the new file, alone
# or as ssh_whole reads it. It reads ws_tmpfile, which lib.sh's ws_tmp sets.
ssh_files() {
    conf=$HOME/.ssh/config.d/ws
    if [ -e "$conf" ] || [ -L "$conf" ]; then
        first=$(head -n 1 "$conf" 2>/dev/null) || first=
        case $first in
        "$header"*) ;;
        *) ws_die "$conf is not mac/setup.sh's: move it away, as the script writes its own there" ;;
        esac
    fi
    at=$(ws_include_ok "$HOME/.ssh/config" config.d/ws) || exit 1
    patterns=$(ws_ignore_unknown "$HOME/.ssh/config" UseKeychain)
    ws_identity "$conf"
    ws_tmp
    new=$ws_tmpfile
    ssh_config "$patterns" >"$new"
    if ! err=$(ssh -G -F "$new" ws-ssh 2>&1 >/dev/null) ||
        ! err=$(ssh -G -F "$new" ws 2>&1 >/dev/null); then
        rm -f "$new"
        ws_die "ssh refuses the config it would write, as $err;" \
            "Match localnetwork needs OpenSSH 9.4"
    fi
    ssh_whole "$new" "${at#first }"
    [ -d "$HOME/.ssh" ] || ws_dir "$HOME/.ssh" 700
    ws_dir "$HOME/.ssh/config.d" 700
    if [ -L "$HOME/.ssh/config.d" ] && ! cmp -s "$conf" "$new" 2>/dev/null; then
        ws_say "warning: ~/.ssh/config.d is a link to $(readlink "$HOME/.ssh/config.d"):" \
            "config.d/ws, with mac.conf's addresses, goes there"
    fi
    ws_put "$conf" 600 <"$new"
    rm -f "$new"
    ws_include_first "$HOME/.ssh/config" config.d/ws
}

# ws_include_at FILE PATH prints where the ssh_config FILE has `Include PATH`, PATH under ~/.ssh:
# "first LINE" where no setting comes before it, "below LINE" where one does, nothing where FILE
# lacks it. The line counts by ~/.ssh/ or the home's full path too, in quotes, before a comment, or
# with other paths after PATH. awk reads the file a link points to.
ws_include_at() {
    awk -v want="$2" -v home="$HOME" '
        {
            line = $0
            sub(/^[ \t]+/, "", line)
            sub(/[ \t\r]+$/, "", line)
        }
        line == "" || line ~ /^#/ { next }
        {
            key = line
            sub(/[ \t=].*/, "", key)
            path = substr(line, length(key) + 1)
            sub(/^[ \t]*=?[ \t]*/, "", path)
            if (substr(path, 1, 1) == "\"") {
                path = substr(path, 2)
                sub(/".*/, "", path)
            } else
                sub(/[ \t].*/, "", path)
            if (index(path, "~/.ssh/") == 1)
                path = substr(path, 8)
            else if (index(path, home "/.ssh/") == 1)
                path = substr(path, length(home) + 7)
        }
        tolower(key) == "include" && path == want {
            print (settings ? "below " : "first ") NR
            exit
        }
        { settings++ }
    ' "$1" || ws_die "cannot read $1"
}

# ws_include_ok FILE PATH stops where ws_include_first could not put `Include PATH` first in the
# ssh_config FILE: where the line sits below a setting, or FILE is a link that lacks it. Else it
# prints what ws_include_at does, and nothing where FILE is missing.
ws_include_ok() {
    [ -e "$1" ] || [ -L "$1" ] || return 0
    ws_at=$(ws_include_at "$1" "$2") || exit 1
    case $ws_at in
    below*)
        ws_die "line ${ws_at#below } of $1 is the Include of $2, below other settings, which ssh" \
            "would take first: move it to the top"
        ;;
    '')
        if [ -L "$1" ]; then
            ws_die "$1 is a link to $(readlink "$1"): put 'Include $2' at the top of that file"
        fi
        ;;
    esac
    echo "$ws_at"
}

# ws_include_first FILE PATH puts `Include PATH` above every setting of the ssh_config FILE, as ssh
# keeps the first value it reads for a key. Where FILE is missing, it makes it. Where FILE lacks the
# line, it adds it at the top, keeping the old file as FILE.TIME~. It stops where ws_include_ok
# does. It reads ws_tmpfile, which lib.sh's ws_tmp sets.
ws_include_first() {
    if [ ! -e "$1" ] && [ ! -L "$1" ]; then
        ws_put "$1" 600 <<EOF
Include $2
EOF
        return 0
    fi
    ws_at=$(ws_include_ok "$1" "$2") || exit 1
    [ -z "$ws_at" ] || return 0
    if ! ws_checking; then
        ws_backup=$1.$(date +%Y%m%d%H%M%S)~
        cp -p "$1" "$ws_backup" || ws_die "cannot keep $1 as $ws_backup"
        ws_say "kept the old $1 as $ws_backup"
    fi
    ws_tmp
    ws_top=$ws_tmpfile
    { printf 'Include %s\n\n' "$2" && cat "$1"; } >"$ws_top" || ws_die "cannot write $ws_top"
    ws_put "$1" "$(ws_mode "$1")" <"$ws_top"
    rm -f "$ws_top"
}

# ws_ignore_unknown FILE FIRST prints FIRST and the patterns of every IgnoreUnknown of the
# ssh_config FILE, in any block, each once, joined by commas. ssh keeps the first IgnoreUnknown it
# reads, so the one in a file included above FILE's settings must carry them all.
ws_ignore_unknown() {
    ws_from=/dev/null
    [ ! -f "$1" ] || ws_from=$1
    awk -v first="$2" '
        function add(pattern,  lower) {
            lower = tolower(pattern)
            if (pattern == "" || lower in seen)
                return
            seen[lower] = 1
            out = out (out == "" ? "" : ",") pattern
        }
        BEGIN { add(first) }
        {
            line = $0
            sub(/^[ \t]+/, "", line)
            key = line
            sub(/[ \t=].*/, "", key)
        }
        tolower(key) == "ignoreunknown" {
            patterns = substr(line, length(key) + 1)
            sub(/^[ \t]*=?[ \t]*/, "", patterns)
            sub(/[ \t]#.*/, "", patterns)
            sub(/[ \t\r]+$/, "", patterns)
            gsub(/"/, "", patterns)
            n = split(patterns, each, /[, \t]+/)
            for (i = 1; i <= n; i++)
                add(each[i])
        }
        END { print out }
    ' "$ws_from" || ws_die "cannot read $1"
}

# ws_own_hosts FILE NAME... warns of each line of the ssh_config FILE that starts a Host or Match
# block for one of the NAMEs, and names those it matches. Its settings come after those of the
# Include at the file's top.
ws_own_hosts() {
    [ -f "$1" ] || return 0
    ws_file=$1
    shift
    awk -v names=" $* " '
        {
            key = tolower($1)
            if (key != "host" && key != "match") next
            hits = ""
            for (i = 2; i <= NF; i++)
                if (index(names, " " $i " ") && !index(" " hits " ", " " $i " "))
                    hits = hits (hits == "" ? "" : " and ") $i
            if (hits != "") print NR ":" hits ":" $0
        }
    ' "$ws_file" | while IFS=: read -r ws_n ws_names ws_line; do
        ws_say "warning: line $ws_n of $ws_file, $ws_line, sets its own values for $ws_names:" \
            "those in config.d/ws, read first, win where both set a key"
    done
}
