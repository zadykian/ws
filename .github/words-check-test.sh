#!/bin/sh
# Tries words-check.sh on a throwaway repository, with stand-in words: a commit for each case, and
# the title and body. It needs no secret. CI's lint job runs it.

set -eu

here=$(cd "$(dirname "$0")" && pwd)
script=$here/words-check.sh
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
failed=0

# One word and one phrase, with a _ in the entry, normalized as the job normalizes the secret.
printf '%s\n' 'acmeword' 'foo bar' 'baz qux' >"$tmp/list"

export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null
export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@example.com
export GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@example.com
repo=$tmp/repo
git init -q "$repo"
cd "$repo"
git config commit.gpgsign false
echo clean >base.txt
git add base.txt
git commit -q -m 'the base'
base=$(git rev-parse HEAD)

# event KEY BASE HEAD BRANCH TITLE BODY writes the event file that the job reads.
event() {
    jq -n --arg k "$1" --arg b "$2" --arg h "$3" --arg r "$4" --arg t "$5" --arg y "$6" \
        '{($k): {base: {sha: $b}, head: {sha: $h, ref: $r}, title: $t, body: $y}}' >"$tmp/event"
}

# run EXPECTED NAME [BRANCH TITLE BODY] runs the script on base..HEAD and compares the status.
# The output must hold none of the stand-in words, and a hit needs the output to say where.
run() {
    want=$1 name=$2
    rc=0
    event pull_request "$base" "$(git rev-parse HEAD)" "${3-feature}" "${4-a title}" "${5-a body}"
    sh "$script" "$tmp/list" pr "$tmp/event" >"$tmp/out" 2>&1 || rc=$?
    if [ "$rc" != "$want" ]; then
        echo "FAIL $name: status $rc, want $want"
        failed=1
    fi
    if grep -qiE 'acmeword|foo|bar|baz|qux' "$tmp/out"; then
        echo "FAIL $name: the output holds a stand-in word"
        failed=1
    fi
    if [ "$want" = 1 ] && ! grep -q "$EXPECT" "$tmp/out"; then
        echo "FAIL $name: the output lacks $EXPECT"
        failed=1
    fi
    git reset -q --hard "$base"
}

# commit_file FILE TEXT MESSAGE commits FILE holding TEXT.
commit_file() {
    printf '%s\n' "$2" >"$1"
    git add -- "$1"
    git commit -q -m "$3"
}

EXPECT='blob .* line 2'
printf 'one\nAcmeWord here\n' >f.txt
git add f.txt && git commit -q -m 'add'
run 1 'an added line'

EXPECT='blob .* line 1'
commit_file f.txt 'Foo_Bar' 'add'
run 1 'an added line, a _ for a space'

EXPECT='blob .* line 1'
commit_file f.txt 'x-acmeword-y' 'add'
run 1 'an added line, a - beside a word'

EXPECT='a file name'
commit_file acmeword.txt 'clean' 'add'
run 1 'a file name'

EXPECT='a file name'
commit_file ok.txt 'clean' 'add'
git mv ok.txt foo_bar.txt && git commit -q -m 'rename'
run 1 'a renamed file'

EXPECT='message line 2'
commit_file f.txt 'clean' "$(printf 'a subject\nbaz_qux in the body')"
run 1 'a message'

commit_file f.txt 'acmewords and xacmeword and acmeword2' 'a clean message'
run 0 'a word inside a longer word'

commit_file f.txt 'clean' 'add'
run 0 'a clean commit'

# A word removed again in a later commit is still in the history.
EXPECT='blob'
commit_file f.txt 'acmeword' 'add'
commit_file f.txt 'clean' 'remove'
run 1 'a word in an earlier commit'

EXPECT='branch name'
commit_file f.txt 'clean' 'add'
run 1 'a branch name' foo_bar

EXPECT='title'
run 1 'a title' feature 'The ACMEWORD'

EXPECT='body line 2'
run 1 'a body' feature 'a title' "$(printf 'fine\nsay foo bar')"

# An issue has a title and a body alone, and needs no repository.
EXPECT='title'
rc=0
event issue '' '' '' 'baz qux' fine
sh "$script" "$tmp/list" issue "$tmp/event" >"$tmp/out" 2>&1 || rc=$?
if [ "$rc" != 1 ] || ! grep -q "$EXPECT" "$tmp/out"; then
    echo 'FAIL an issue title'
    failed=1
fi
rc=0
event issue '' '' '' fine fine
sh "$script" "$tmp/list" issue "$tmp/event" >"$tmp/out" 2>&1 || rc=$?
[ "$rc" = 0 ] || {
    echo 'FAIL a clean issue'
    failed=1
}

# A bad range is an error, status 2.
rc=0
event pull_request "$base" nothere b t b
sh "$script" "$tmp/list" pr "$tmp/event" >"$tmp/out" 2>&1 || rc=$?
[ "$rc" = 2 ] || {
    echo "FAIL a bad range: status $rc"
    failed=1
}

if [ "$failed" = 0 ]; then echo "words-check-test: ok"; fi
exit "$failed"
