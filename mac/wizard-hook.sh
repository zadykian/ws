# shellcheck shell=sh
# The wizard's stages for another repository, which mac.conf's WS_HOOK names as OWNER/REPO. Hook on
# ws runs its apply.sh as root on ws. The wizard's files script, wizard-remote-files.sh, clones the
# repository in /root/repository/REPO there, by root's gh login, where it is missing, and never
# pulls it. apply.sh then runs by plain ssh, its stdin the Keychain item that WS_HOOK_SECRET names,
# as security prints it, or else empty. A stamp on ws marks a run of it that exited 0, so that a
# second run of the wizard leaves it alone. Hook on the Mac clones the repository in
# ~/repository/REPO by gh's login there, with its hooks and its Brewfile's formulae. It copies ws's
# word list too, with or without WS_HOOK, where the Mac's differs. Where git lacks gh's credential
# command, the closing list names gh auth setup-git. wizard.sh sources it, and runs the stages after
# the Tunnel. It calls the functions of lib.sh, setup-lib.sh, wizard-gh.sh, wizard-host.sh and
# wizard-ws.sh, and wizard.sh's by_hand. It reads wizard.sh's dir, lock, update and WS_CONF, and
# wizard-host.sh's root_in.

# wshook_ran becomes 1 once the stage ran apply.sh in this run.
wshook_ran=0
# The Mac's word list, which ws's hooks read in its clone, as another repository's may in theirs.
hook_words=$HOME/.config/ws/forbidden-words

# remote_files OP [ARG...] runs the files script's OP as root on ws, by host_ssh, as remote does.
# shellcheck disable=SC2154 # dir is wizard.sh's
remote_files() {
    host_ssh -o BatchMode=yes ws-ssh sh -s -- "$@" \
        <"$dir/wizard-remote-files.sh"
}

# wshook_after prints the stage that the clone waits on, where that isn't done. That is ws's run,
# which installs gh on ws, then gh on ws, whose login clones the repository.
wshook_after() {
    if ! run_done; then
        echo "ws's run"
    elif ! wsgh_done; then
        echo "gh on ws"
    fi
}

# wshook_done succeeds where mac.conf names no repository, or where ws has the stamp of its
# apply.sh, which names it as OWNER/REPO. Under --update, only once the stage ran apply.sh in this
# run. It reads ws only where root logged in there in this run.
# shellcheck disable=SC2154 # update is wizard.sh's, root_in wizard-host.sh's
wshook_done() {
    ws_conf "$WS_CONF"
    [ -n "$WS_HOOK" ] || return 0
    [ "$update" = 0 ] || [ "$wshook_ran" = 1 ] || return 1
    [ "$root_in" = 1 ] && remote_read &&
        case " $(remote_get hook) " in *" $WS_HOOK "*) ;; *) false ;; esac
}

# wshook_do runs the repository's apply.sh as root on ws, by plain ssh, never ssh -t, at which such
# a script could prompt and echo its stdin. The files script's clone first clones the repository
# where ws lacks it, and removes its stamp, which its stamp op writes again once apply.sh exits 0.
# First, security must read the item into /dev/null, so that an item it can't read stops the stage
# before ssh. --check says what it would do.
# shellcheck disable=SC2154 # lock is wizard.sh's
wshook_do() {
    wshook_apply=/root/repository/${WS_HOOK#*/}/apply.sh
    wshook_in=${WS_HOOK_SECRET:+the Keychain item $WS_HOOK_SECRET}
    wshook_item || return 1
    if ws_checking; then
        ws_say "Hook on ws: would clone $WS_HOOK into ${wshook_apply%/*} on ws, where missing," \
            "then run its apply.sh there, its stdin ${wshook_in:-empty}"
        return 0
    fi
    [ -z "$WS_HOOK_SECRET" ] ||
        security find-generic-password -s "$WS_HOOK_SECRET" -w >/dev/null 2>"$lock/hook.err" ||
        wshook_unread
    ws_say "Hook on ws: $WS_HOOK's apply.sh runs as root on ws, its stdin ${wshook_in:-empty}"
    # shellcheck disable=SC2034 # wizard-ws.sh's remote_read reads it
    remote_got=0
    remote_files clone "$WS_HOOK" >&2 || return 1
    wshook_run || return 1
    remote_files stamp "$WS_HOOK" >&2 || return 1
    wshook_ran=1
}

# wshook_item adds the Keychain item that WS_HOOK_SECRET names, where the Keychain lacks it, by
# security, which asks for its secret twice at the terminal. The wizard never reads the secret.
wshook_item() {
    [ -n "$WS_HOOK_SECRET" ] || return 0
    ! security find-generic-password -s "$WS_HOOK_SECRET" >/dev/null 2>&1 || return 0
    wshook_add="security add-generic-password -a $USER -s $WS_HOOK_SECRET -w"
    if ws_checking; then
        ws_say "Hook on ws: would run $wshook_add, which asks for the item's secret at the terminal"
        return 0
    fi
    [ -t 0 ] || ws_die "Hook on ws: the Keychain lacks the item $WS_HOOK_SECRET, and no terminal" \
        "is there for $wshook_add: run it, then the wizard again"
    ws_say "Hook on ws: $wshook_add asks for the secret of $WS_HOOK's apply.sh, twice"
    security add-generic-password -a "$USER" -s "$WS_HOOK_SECRET" -w ||
        ws_die "Hook on ws: $wshook_add failed, as above; run the wizard again"
}

# wshook_unread stops the stage where security can't read the item, as in a locked Keychain, or
# where the user denied macOS's prompt for it. It names security's last line of error, where it
# printed one in hook.err.
# shellcheck disable=SC2154 # lock is wizard.sh's
wshook_unread() {
    wshook_why=$(tail -n 1 "$lock/hook.err" 2>/dev/null)
    ws_die "Hook on ws: security can't read the item $WS_HOOK_SECRET from the Keychain, so" \
        "apply.sh didn't run; unlock or allow it, then run the wizard" \
        "again${wshook_why:+: $wshook_why}"
}

# wshook_guard is the script that ssh runs on ws as sh -c GUARD APPLY, where WS_HOOK_SECRET names an
# item. It runs APPLY, apply.sh, on its stdin, but exits 3 where that is empty. So apply.sh never
# runs without its secret, even where the read for ssh fails, as a second prompt can deny it.
# shellcheck disable=SC2016 # sh expands it on ws
wshook_guard='IFS= read -r l || [ -n "$l" ] || exit 3; { printf %s "$l"; echo; cat; } | exec "$0"'

# wshook_run runs apply.sh on ws, its stdin the item's secret, as security reads it a second time,
# through wshook_guard, or /dev/null without an item. It fails where apply.sh or ssh does. Where
# security fails once ssh reached ws, it stops, as wshook_unread does.
# shellcheck disable=SC2154 # lock is wizard.sh's
wshook_run() {
    if [ -z "$WS_HOOK_SECRET" ]; then
        host_ssh -o BatchMode=yes ws-ssh "$wshook_apply" </dev/null >&2
        return
    fi
    rm -f "$lock/hook.status"
    wshook_status=0
    wshook_read | host_ssh -o BatchMode=yes ws-ssh \
        "sh -c '$wshook_guard'" "$wshook_apply" >&2 || wshook_status=$?
    [ ! -f "$lock/hook.status" ] || [ "$wshook_status" = 255 ] || wshook_unread
    return "$wshook_status"
}

# wshook_read prints the item's secret for ssh, as security reads it. Where security fails, its
# status goes in hook.status, and its error in hook.err.
wshook_read() {
    security find-generic-password -s "$WS_HOOK_SECRET" -w 2>"$lock/hook.err" ||
        echo $? >"$lock/hook.status"
}

# wshook_fail says what to do where the clone or apply.sh failed, or ws still lacks the stamp.
wshook_fail() {
    echo "the clone of $WS_HOOK on ws, or its apply.sh, failed, as above; mend what they printed," \
        "then run the wizard again, which runs apply.sh again"
}

# hook_after prints the stage that this one waits on, where that isn't done: Root's key, without
# which the wizard reads no word list on ws. Where mac.conf names a repository, then Hook on ws,
# whose apply.sh may write that list, and gh on the Mac, whose login clones the repository.
hook_after() {
    if [ "$root_in" != 1 ]; then
        echo "Root's key"
    elif [ -n "$WS_HOOK" ] && ! wshook_done; then
        echo "Hook on ws"
    elif [ -n "$WS_HOOK" ] && ! gh_done; then
        echo "gh on the Mac"
    fi
}

# hook_done succeeds where mac.conf names no repository, or where its clone is in ~/repository/REPO,
# with core.hooksPath at its .githooks and its Brewfile's formulae installed, where it has them. The
# Mac's word list must be ws's too, where ws has one. It reads ws only where root logged in there.
hook_done() {
    ws_conf "$WS_CONF"
    hook_dir=$HOME/repository/${WS_HOOK#*/}
    [ -z "$WS_HOOK" ] || { [ -e "$hook_dir/.git" ] && hook_origin && hook_hooks && hook_brew; } ||
        return 1
    [ "$root_in" = 1 ] && hook_sums && { [ "$hook_ws" = - ] || [ "$hook_ws" = "$hook_mac" ]; }
}

# hook_sums sets hook_ws to git's hash of ws's word list, as the files script prints it, and
# hook_mac to the Mac's, each - where there is none. A Mac's list that the user can't read counts as
# none: hook_copy replaces it, or stops at a link. git hashes from /, as in a repository it takes
# that repository's hash, which may be SHA-256. It fails where ssh or the files script does, their
# messages in words.err.
# shellcheck disable=SC2154 # lock is wizard.sh's
hook_sums() {
    hook_ws=$(remote_files words 2>"$lock/words.err") && [ -n "$hook_ws" ] || return 1
    hook_mac=-
    [ ! -f "$hook_words" ] || [ ! -r "$hook_words" ] ||
        hook_mac=$(git -C / hash-object --stdin <"$hook_words") ||
        ws_die "Hook on the Mac: git can't hash the Mac's word list, $hook_words, as above"
}

# hook_copy copies ws's word list over the Mac's, mode 600, where ws has one and the Mac's differs.
# The list goes through the lock, whose files are the user's alone, and is never printed. --check
# says it would copy it, and shows no diff. Where the Mac's is a link, it stops, under --check too,
# naming its target, as ws_put would replace the link.
hook_copy() {
    hook_sums || ws_die "Hook on the Mac: the files script read no word list on ws:" \
        "$(tail -n 1 "$lock/words.err")"
    [ "$hook_ws" != - ] && [ "$hook_ws" != "$hook_mac" ] || return 0
    [ ! -L "$hook_words" ] || ws_die "Hook on the Mac: $hook_words is a link to" \
        "$(readlink "$hook_words"), which the wizard leaves, and ws's word list differs: remove" \
        "the link, or copy root's ~/.config/ws/forbidden-words on ws to that file, then run the" \
        "wizard again"
    if ws_checking; then
        ws_say "Hook on the Mac: would copy ws's word list to $hook_words, mode 600"
        return 0
    fi
    (umask 077 && remote_files words "$hook_ws" >"$lock/words") &&
        [ "$(git -C / hash-object --stdin <"$lock/words")" = "$hook_ws" ] ||
        ws_die "Hook on the Mac: ws's word list came short or changed, as above; run the" \
            "wizard again"
    ws_put "$hook_words" 600 <"$lock/words"
    rm -f "$lock/words"
}

# hook_origin succeeds where the clone's origin names WS_HOOK on GitHub, as the files script reads
# the clone on ws: by https or ssh, as gh sets it, letter case aside.
hook_origin() {
    hook_url=$(git -C "$hook_dir" remote get-url origin 2>/dev/null) || hook_url=
    case $(printf '%s\n' "${hook_url%.git}" | tr '[:upper:]' '[:lower:]') in
    *[/@]github.com[/:]"$(printf '%s\n' "${WS_HOOK%.git}" | tr '[:upper:]' '[:lower:]')") ;;
    *) return 1 ;;
    esac
}

# hook_hooks succeeds where the clone has no .githooks, or core.hooksPath names it, by an absolute
# path, as in ws's clone (docs/checks.md, The hooks).
hook_hooks() {
    [ ! -d "$hook_dir/.githooks" ] ||
        [ "$(git -C "$hook_dir" config core.hooksPath)" = "$hook_dir/.githooks" ]
}

# hook_brew succeeds where the clone has no Brewfile, or Homebrew has all it names. An outdated
# formula counts as there, as the stage upgrades nothing. The check runs without Homebrew's
# auto-update, which brew bundle would run first a day after the last, writing under --check too.
hook_brew() {
    [ ! -f "$hook_dir/Brewfile" ] || HOMEBREW_NO_AUTO_UPDATE=1 \
        brew bundle check --no-upgrade --file="$hook_dir/Brewfile" >/dev/null 2>&1
}

# hook_do clones the repository by gh's login, where nothing is there, and stops where the path
# holds no clone, or another repository's. It points core.hooksPath at the clone's .githooks, and
# runs brew bundle, which installs what the Brewfile names and Homebrew lacks. --check says what it
# would run. First it copies ws's word list, where the Mac's differs.
hook_do() {
    hook_copy
    [ -n "$WS_HOOK" ] || return 0
    if [ ! -e "$hook_dir" ] && [ ! -L "$hook_dir" ]; then
        if ws_checking; then
            ws_say "Hook on the Mac: would run gh repo clone $WS_HOOK $hook_dir, then set its" \
                "core.hooksPath and run brew bundle, where it has .githooks or a Brewfile"
            return 0
        fi
        hook_run gh repo clone "$WS_HOOK" "$hook_dir" -- -q
    fi
    [ -e "$hook_dir/.git" ] || ws_die "Hook on the Mac: $hook_dir holds no clone: move it aside"
    hook_origin || ws_die "Hook on the Mac: $hook_dir is not $WS_HOOK's clone, its origin" \
        "'$hook_url': move it aside"
    hook_hooks || hook_run git -C "$hook_dir" config core.hooksPath "$hook_dir/.githooks"
    hook_brew || hook_run brew bundle --no-upgrade --file="$hook_dir/Brewfile"
}

# hook_run CMD... runs CMD, which it names first, and stops where CMD fails. --check says it would.
hook_run() {
    if ws_checking; then
        ws_say "Hook on the Mac: would run $*"
        return 0
    fi
    ws_say "Hook on the Mac: running $*"
    "$@" >&2 || ws_die "Hook on the Mac: $* failed, as above"
}

# hook_fail says what to do where the Mac still lacks the clone, its hooks, its Brewfile's formulae
# or ws's word list.
hook_fail() {
    echo "the clone of WS_HOOK's repository on the Mac, its hooks, its Brewfile's formulae or" \
        "ws's word list failed, as above; mend what they printed, then run the wizard again"
}

# hook_hand adds gh auth setup-git to what stays by hand, where git's config names no credential
# command of gh's for github.com, as setup.sh's gitconfig does where mac.conf gives git a name.
# Through that command, pushes from the Mac's clones take gh's login.
hook_hand() {
    git config --global --includes --get-all credential.https://github.com.helper 2>/dev/null |
        grep -q ' auth git-credential$' || by_hand "Pushes from the Mac's clones to GitHub:" \
        "gh auth setup-git, which has git take gh's login (docs/mac-git.md)."
}
