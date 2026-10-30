# shellcheck shell=sh
# ws's word list on the Mac, which the hooks of ws's clone and of another repository's read
# (docs/checks.md, The hooks). wizard-hook.sh's Hook on the Mac copies it from ws, by words_copy,
# where the Mac's differs. wizard.sh sources it. It calls the functions of lib.sh and
# wizard-hook.sh's remote_files, and reads wizard.sh's lock and wizard-host.sh's root_in.

# The Mac's word list, which ws's hooks read in its clone, as another repository's may in theirs.
words_file=$HOME/.config/ws/forbidden-words

# words_sums sets words_ws to git's hash of ws's word list, as the files script prints it, and
# words_mac to the Mac's, each - where there is none. A Mac's list that the user can't read counts
# as none: words_copy replaces it, or stops at a link. git hashes from /, as in a repository it
# takes that repository's hash, which may be SHA-256. It fails where ssh or the files script does,
# their messages in words.err.
# shellcheck disable=SC2154 # lock is wizard.sh's
words_sums() {
    words_ws=$(remote_files words 2>"$lock/words.err") && [ -n "$words_ws" ] || return 1
    words_mac=-
    [ ! -f "$words_file" ] || [ ! -r "$words_file" ] ||
        words_mac=$(git -C / hash-object --stdin <"$words_file") ||
        ws_die "Word list: git can't hash the Mac's word list, $words_file, as above"
}

# words_copy copies ws's word list over the Mac's, mode 600, where ws has one and the Mac's differs.
# The list goes through the lock, whose files are the user's alone, and is never printed. --check
# says it would copy it, and shows no diff. Where the Mac's is a link, it stops, under --check too,
# naming its target, as ws_put would replace the link.
words_copy() {
    words_sums || ws_die "Word list: the files script read no word list on ws:" \
        "$(tail -n 1 "$lock/words.err")"
    [ "$words_ws" != - ] && [ "$words_ws" != "$words_mac" ] || return 0
    [ ! -L "$words_file" ] || ws_die "Word list: $words_file is a link to" \
        "$(readlink "$words_file"), which the wizard leaves, and ws's word list differs: remove" \
        "the link, or copy root's ~/.config/ws/forbidden-words on ws to that file, then run the" \
        "wizard again"
    if ws_checking; then
        ws_say "Word list: would copy ws's word list to $words_file, mode 600"
        return 0
    fi
    (umask 077 && remote_files words "$words_ws" >"$lock/words") &&
        [ "$(git -C / hash-object --stdin <"$lock/words")" = "$words_ws" ] ||
        ws_die "Word list: ws's word list came short or changed, as above; run the" \
            "wizard again"
    ws_put "$words_file" 600 <"$lock/words"
    rm -f "$lock/words"
}
