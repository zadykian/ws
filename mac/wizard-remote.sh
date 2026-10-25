#!/bin/sh
# The wizard's remote script on ws, which the Mac's wizard sends to root there over ssh. So it is
# the Mac's clone's, whatever ws's clone holds, or where ws has none:
#
#   ssh ws-ssh sh -s -- OP [ARG...] <mac/wizard-remote.sh
#
# - status prints what the wizard reads of ws, a line each, as NAME VALUE, with - for a value that
#   is missing. run is the commit and status of the last full run, and last the unit and status of
#   the last run. active is the run's unit that runs now. mac.pub is the Mac's key that wg0's peer
#   takes, and wg0 is wg0's own key. gh is ok, none, login, scope or key, below. claude is ok, none
#   or login. fans is refused where Secure Boot refuses nct6687, the fans' driver, else ok.
# - run [--update] [ARG...] runs bootstrap.sh ARGs in a unit of its own, ws-wizard-STAMP, which goes
#   on where ssh drops, or follows such a unit that runs already. It prints the unit's log until it
#   ends, and exits with the run's status. --update first pulls a clean clone on main.
#
# Everything is in functions that the last line calls, so that a script cut short runs nothing.

set -eu

# The wizard's stamps, ws's clone, the Mac's key for wg0, root's claude, ws's signing key, and the
# fans' driver as the kernel loaded it. Each is a line NAME=PATH, which CI's stand-in for ws roots
# in a directory of its own.
state=/var/lib/ws-wizard
dir=/root/repository/ws
peer=/etc/wireguard/mac.pub
claude=/root/.local/bin/claude
signing=/root/.ssh/id_ed25519_signing.pub
module=/sys/module/nct6687

# say prints a message of the script's on stderr.
say() {
    printf 'wizard-remote.sh: %s\n' "$*" >&2
}

# die prints a message of the script's and exits with status 1.
die() {
    say "$*"
    exit 1
}

# first prints the first line of its stdin, or - where it has none.
first() {
    awk 'NR == 1 { print; found = 1 } END { if (!found) print "-" }'
}

# stamp FILE prints the two words of the stamp FILE, or - where it has none.
stamp() {
    awk 'NF == 2 { print; exit }' "$1" 2>/dev/null | first
}

# active prints the run's unit that is active or activating, where there is one.
active() {
    systemctl list-units --type=service --state=active,activating --no-legend --plain \
        'ws-wizard-*' 2>/dev/null | awk '{ sub(/\.service$/, "", $1); print $1; exit }'
}

# gh_state prints none without gh, and login without a login to github.com, or where its list of
# signing keys fails but for a 404. scope is that 404, as the login lacks the scope. key is where
# ws's signing key is missing or not listed, and ok where it is.
gh_state() {
    command -v gh >/dev/null || { echo none && return; }
    gh auth status -h github.com >/dev/null 2>&1 || { echo login && return; }
    [ -f "$signing" ] || { echo key && return; }
    if ! keys=$(gh api --paginate user/ssh_signing_keys --jq '.[].key' 2>&1); then
        case $keys in *'HTTP 404'*) echo scope ;; *) echo login ;; esac
    elif printf '%s\n' "$keys" | grep -qxF -e "$(cut -d ' ' -f 1-2 "$signing")"; then
        echo ok
    else
        echo key
    fi
}

# status prints what the wizard reads of ws, as the header says.
status() {
    echo "run $(stamp "$state/run")"
    echo "last $(stamp "$state/last")"
    echo "active $(active | first)"
    echo "mac.pub $(head -n 1 "$peer" 2>/dev/null | first)"
    echo "wg0 $(wg show wg0 public-key 2>/dev/null | first)"
    echo "gh $(gh_state)"
    if [ ! -x "$claude" ]; then
        echo 'claude none'
    elif "$claude" auth status >/dev/null 2>&1; then
        echo 'claude ok'
    else
        echo 'claude login'
    fi
    if modinfo nct6687 >/dev/null 2>&1 && [ ! -d "$module" ] &&
        mokutil --sb-state 2>/dev/null | grep -q 'SecureBoot enabled'; then
        echo 'fans refused'
    else
        echo 'fans ok'
    fi
}

# pull pulls the clone by fast-forward, where it is on main with nothing uncommitted, untracked
# files aside. Otherwise it warns, and leaves it as it is.
pull() {
    [ -e "$dir/.git" ] || return 0
    if [ "$(git -C "$dir" symbolic-ref -q --short HEAD)" != main ] ||
        [ -n "$(git -C "$dir" status --porcelain --untracked-files=no)" ]; then
        say "warning: leaves $dir as it is, not on main or not clean: pull it by hand"
        return 0
    fi
    head=$(git -C "$dir" rev-parse --short HEAD)
    git -C "$dir" pull -q --ff-only || die "git pull --ff-only failed in $dir; mend the clone"
    [ "$(git -C "$dir" rev-parse --short HEAD)" = "$head" ] ||
        say "pulled $dir to $(git -C "$dir" rev-parse --short HEAD)"
}

# job prints the script of a run's unit, which systemd-run runs with UNIT DIR STATE [ARG...]. It
# runs bootstrap.sh ARGs from the clone in DIR, or from the README's download where DIR holds none.
# Its last steps write STATE/last as UNIT and the status, and, without ARGs, STATE/run as the
# clone's commit and the status, each by a rename. A full run removes STATE/run first, as it starts.
job() {
    cat <<'EOF'
unit=$1 dir=$2 state=$3 full=$(($# - 3))
shift 3
[ "$full" != 0 ] || rm -f "$state/run"
if [ -e "$dir/.git" ]; then
    sh "$dir/bootstrap.sh" "$@"
else
    curl -fsSL --retry 3 -o "$state/bootstrap.sh" \
        https://raw.githubusercontent.com/zadykian/ws/main/bootstrap.sh &&
        sh "$state/bootstrap.sh" "$@"
fi
status=$?
if [ "$full" = 0 ]; then
    sha=$(git -C "$dir" rev-parse HEAD 2>/dev/null) || sha=-
    echo "$sha $status" >"$state/run.new" && mv -f "$state/run.new" "$state/run"
fi
echo "$unit $status" >"$state/last.new" && mv -f "$state/last.new" "$state/last"
exit "$status"
EOF
}

# run starts a run's unit, or follows the one that runs, as the header says. The unit takes root's
# HOME, which systemd gives a unit only with a user, and the locale that Ansible needs.
run() {
    update=0
    [ "${1-}" != --update ] || { update=1 && shift; }
    unit=$(active)
    if [ -n "$unit" ]; then
        say "following $unit, which runs on ws already"
    else
        [ "$update" = 0 ] || pull
        mkdir -p "$state"
        job >"$state/job.new"
        mv -f "$state/job.new" "$state/job"
        unit=ws-wizard-$(date +%Y%m%d%H%M%S)-$$
        systemd-run --quiet --unit="$unit" --collect --expand-environment=no \
            --description="ws's playbook, for the Mac's wizard" --setenv=LC_ALL=C.UTF-8 \
            --setenv=HOME=/root /bin/sh "$state/job" "$unit" "$dir" "$state" "$@" ||
            die "systemd-run cannot start $unit"
        say "started $unit, which goes on where ssh drops: journalctl -u $unit shows its log"
    fi
    follow "$unit"
}

# follow UNIT prints UNIT's log every 2 s while it runs, and once more a second after it ends. It
# exits with the status that UNIT's job wrote in last, or 1 where the job ended before it wrote one.
follow() {
    cursor=$(mktemp -d)
    trap 'rm -rf "$cursor"' EXIT
    trap 'exit 1' HUP INT TERM
    while running "$1"; do
        journal "$1"
        sleep 2
    done
    sleep 1
    journal "$1"
    last=$(stamp "$state/last")
    [ "${last% *}" = "$1" ] ||
        die "$1 ended before its run wrote its status, as where it was stopped: run it again"
    [ "${last#* }" = 0 ] || say "$1's run exited ${last#* }: journalctl -u $1 shows its log"
    exit "${last#* }"
}

# running succeeds while UNIT is active, or on its way in or out.
running() {
    systemctl show -p ActiveState --value "$1" | grep -qxE "active|activating|deactivating"
}

# journal prints the lines of UNIT's log that follow those it printed before, as its cursor marks.
journal() {
    journalctl -q -u "$1" -o cat --no-pager --cursor-file="$cursor/at" || true
}

main() {
    [ "$(id -u)" = 0 ] || die "it runs as root on ws, as ssh ws-ssh logs in"
    case ${1-} in
    status) status ;;
    run) shift && run "$@" ;;
    *) die "unknown op '${1-}'; usage: wizard-remote.sh status | run [--update] [ARG...]" ;;
    esac
}

# Nothing it runs reads stdin, which under sh -s is the rest of this script. The braces make a
# script cut short a syntax error, rather than main run.
{ main "$@" </dev/null; }
