#!/bin/sh
# Checks a pull request's or an issue's text for the words of the local list, as the hooks in
# .githooks check a commit. CI's words job runs it, and words-check-test.sh tries it.
#
#   words-check.sh LIST pr EVENT
#   words-check.sh LIST issue EVENT
#
# EVENT is the workflow's event file, $GITHUB_EVENT_PATH. The title, body and branch name are read
# from it and never pass through the environment, as a step's env is printed in the log.
#
# LIST holds one word or phrase a line, each _ already a space, blank lines and comments gone. A
# word matches case-insensitively where no letter or digit touches it, and _ separates words, as
# the hooks match it.
#
# The output is public, a run's log. It says where a word is and never what: not the word, a
# path, a ref's name or a line's text. So grep's output goes to files, no command traces, and the
# errors of git and grep never reach the log. It prints a count of hits last, and exits 1 on a hit
# and 2 on an error.

set -eu

list=$1
mode=$2
event=$3
# Fd 3 keeps our own errors, as the commands' stderr goes to /dev/null.
exec 3>&2 2>/dev/null
tmp=$(mktemp -d)
fin=0
hits=0
# Any failure not ended by the final exit is an error.
trap 'c=$?; rm -rf "$tmp"; if [ "$fin" = 0 ] && [ "$c" != 0 ]; then echo "words-check: error" >&3; exit 2; fi' EXIT
trap 'exit 2' INT TERM

# scan FILE writes the numbers of FILE's lines that hold a word to $tmp/nums, and fails where none
# does. grep -w counts _ as part of a word, so grep reads a copy of FILE with each _ a space, as
# in LIST. tr keeps every line, so the numbers hold. It reads bytes, in the C locale.
scan() {
    LC_ALL=C tr _ ' ' <"$1" >"$tmp/text"
    rc=0
    grep -aniwF -f "$list" "$tmp/text" >"$tmp/grep" || rc=$?
    [ "$rc" -le 1 ] || exit 2
    [ "$rc" = 0 ] || return 1
    cut -d: -f1 "$tmp/grep" >"$tmp/nums"
}

# report PREFIX WHAT prints WHAT, with each line number of $tmp/nums, and counts the hits.
report() {
    while read -r n; do
        echo "$1$2 line $n"
        hits=$((hits + 1))
    done <"$tmp/nums"
}

# single PREFIX WHAT FILE prints WHAT where FILE, a text of one line, holds a word.
single() {
    if scan "$3"; then
        echo "$1$2"
        hits=$((hits + 1))
    fi
}

# commit SHA checks a commit as the hooks do: the lines it adds against its first parent, the
# names of the files it adds or renames to, and its message. A root commit has the empty tree for
# its parent.
commit() {
    p=$1^
    git rev-parse --verify -q "$p^{commit}" >/dev/null || p=$(git hash-object -t tree /dev/null)
    # The options pin the diff's shape against the config and .gitattributes: the bytes as text,
    # binary files included, and never a textconv's rewrite of them. A line starting with +++
    # names the file only in the header, before the first @@. The blob is the new file's id.
    git -c core.quotePath=false diff --text --no-textconv --no-color --no-ext-diff --no-prefix \
        --full-index -U0 --inter-hunk-context=0 --diff-filter=d "$p" "$1" >"$tmp/diff"
    awk -v where="$tmp/where" '
        /^diff --git / { header = 1; next }
        header && /^index / { split($2, ids, /\.\./); blob = ids[2]; next }
        /^@@ / {
            header = 0
            n = $3
            sub(/^\+/, "", n)
            sub(/,.*/, "", n)
            n += 0
            next
        }
        header { next }
        /^\+/ { print blob " line " n >where; print substr($0, 2); n++ }
    ' "$tmp/diff" >"$tmp/added"
    : >>"$tmp/where"
    if scan "$tmp/added"; then
        # The line of added is the line of where.
        awk 'NR == FNR { hit[$1] = 1; next } FNR in hit { print }' "$tmp/nums" "$tmp/where" \
            >"$tmp/found"
        while read -r line; do
            echo "$1: blob $line"
            hits=$((hits + 1))
        done <"$tmp/found"
    fi

    # -z, as git quotes a name holding a tab, a newline, a quote or a backslash.
    git diff --name-only --diff-filter=ARC -z "$p" "$1" >"$tmp/names0"
    tr '\0' '\n' <"$tmp/names0" >"$tmp/names"
    single "$1: " 'a file name' "$tmp/names"

    git cat-file commit "$1" | sed '1,/^$/d' >"$tmp/message"
    if scan "$tmp/message"; then
        report "$1: " message
    fi
}

# field PATH FILE writes the event's string at PATH to FILE, empty where null.
field() {
    jq -r "($1) // \"\"" "$event" >"$2"
}

case $mode in
pr)
    field .pull_request.base.sha "$tmp/base"
    field .pull_request.head.sha "$tmp/head"
    read -r base <"$tmp/base"
    read -r head <"$tmp/head"
    git rev-list "$base..$head" >"$tmp/commits"
    while read -r c; do
        commit "$c"
    done <"$tmp/commits"
    field .pull_request.head.ref "$tmp/value"
    single '' 'branch name' "$tmp/value"
    prefix=.pull_request
    ;;
issue) prefix=.issue ;;
*) exit 2 ;;
esac
field "$prefix.title" "$tmp/value"
single '' title "$tmp/value"
field "$prefix.body" "$tmp/value"
if scan "$tmp/value"; then
    report '' body
fi

fin=1
echo "words-check: $hits hits"
[ "$hits" = 0 ]
