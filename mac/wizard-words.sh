# shellcheck shell=sh
# ws's word list on the Mac, which the hooks of ws's clone and of another repository's read
# (docs/checks.md, The hooks). wizard-hook.sh's Hook on the Mac copies it from ws, by hook_copy,
# where the Mac's differs. wizard.sh sources it. It calls the functions of lib.sh and
# wizard-hook.sh's remote_files, and reads wizard.sh's lock and wizard-host.sh's root_in.

# The Mac's word list, which ws's hooks read in its clone, as another repository's may in theirs.
hook_words=$HOME/.config/ws/forbidden-words

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
