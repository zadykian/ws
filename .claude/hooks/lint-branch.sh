#!/bin/sh
# Claude Code's Stop hook (.claude/settings.json): runs make lint as Claude ends its turn. It runs
# in each tree that lint-file.sh recorded for the session and that differs from origin/main. Then
# it clears the record, so a turn that wrote no file runs nothing. A failure prints the end of the
# output to stderr with status 2, and Claude goes on; as it stops again, the user sees what still
# fails. A tree whose code the maintainer does not vouch for gets a line to the user.
#
# shellcheck source-path=SCRIPTDIR

set -u

hooks=$(dirname -- "$0")
# shellcheck source=lib.sh
. "$hooks/lib.sh"

# lint runs make lint in the work tree $1, where it belongs to this repository, the maintainer
# vouches for it and it differs from origin/main. A failure goes to $tmp/report and the tree to
# $tmp/failed, and a notice to $tmp/notes. gitleaks scans the branch's own commits, as a secret
# in a stash or in another branch, which every worktree shares, is none of the turn's. stdin is
# /dev/null and the output a file, as ansible-core refuses handles that do not block.
lint() {
    if ! plain "$1" || [ "$(common "$1")" != "$repo" ] || ! cd -- "$1" || [ ! -f tools/files ]; then
        return 0
    fi
    if ! vouched; then
        unvouched "$1" >>"$tmp/notes"
        return 0
    fi
    if git diff --quiet "$base" -- 2>/dev/null &&
        [ -z "$(git ls-files --others --exclude-standard)" ]; then
        return 0
    fi
    if ! make lint BASE="$base" LOG_OPTS="$base..HEAD" </dev/null >"$tmp/out" 2>&1; then
        printf 'make lint fails in %s, ending with:\n' "$1" >>"$tmp/report"
        tail -n 50 "$tmp/out" >>"$tmp/report"
        printf '%s\n' "$1" >>"$tmp/failed"
    fi
}

# again succeeds where Claude stops again after a failure of this hook: stop_hook_active is true.
# A quote in a JSON string is escaped, so only the key matches.
again() {
    printf '%s\n' "$input" | tr '\n' ' ' |
        grep -q '"stop_hook_active"[[:space:]]*:[[:space:]]*true'
}

main() {
    input=$(cat)
    repo=$(common "$hooks")
    [ -n "$repo" ] || return 0
    record=$(record_path "$repo" "$(printf '%s\n' "$input" | field session_id)") || return 0
    [ -f "$record" ] || return 0
    tmp=$(mktemp -d) || return 0
    trap 'rm -rf "$tmp"' EXIT
    # The record is cleared at each stop, so that the next turn lints only what it writes.
    mv -f -- "$record" "$tmp/trees" || return 0

    : >"$tmp/report"
    : >"$tmp/notes"
    : >"$tmp/failed"
    while IFS= read -r tree; do
        lint "$tree" </dev/null
    done <"$tmp/trees"
    if [ -s "$tmp/report" ] && ! again; then
        cat "$tmp/notes" "$tmp/report" >&2
        printf '%s %s\n' 'Fix what this turn broke. Report a failure from before it, such as' \
            "another commit's message or the maintainer's own edits, rather than fix it." >&2
        return 2
    fi
    # Where Claude stops again, this run linted the trees it wrote since the failure. What still
    # fails goes to the user, as a second status 2 would keep Claude from stopping.
    if [ -s "$tmp/failed" ]; then
        printf 'lint-branch.sh: make lint still fails in %s after the fix.\n' \
            "$(paste -s -d ' ' "$tmp/failed")" >>"$tmp/notes"
    fi
    if [ -s "$tmp/notes" ]; then
        notice "$(paste -s -d ' ' "$tmp/notes")"
    fi
}

main "$@"
