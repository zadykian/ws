#!/bin/sh
# Claude Code's PostToolUse hook for Edit and Write (.claude/settings.json): runs make lint's gates
# on the file just written, by its kind. It records the file's tree for lint-branch.sh too. A
# finding goes to stderr with status 2, which Claude sees. A file outside this repository, or one
# git ignores, such as a plan, is passed over. So is a path make cannot take, and a tree whose
# code the maintainer does not vouch for, each with a line to the user. Such a tree is still
# recorded, as a commit later in the turn may vouch for it, and lint-branch.sh checks again.
#
# shellcheck source-path=SCRIPTDIR

set -u

hooks=$(dirname -- "$0")
# shellcheck source=lib.sh
. "$hooks/lib.sh"

# gate runs make's target $1 on the file from the work tree's root, and adds its output to $report
# where it fails. Its stdin is /dev/null and its output a file, as ansible-core refuses handles
# that do not block.
gate() {
    if ! make -s --no-print-directory "$1" FILES="$path" </dev/null >"$tmp/out" 2>&1; then
        printf 'make %s FILES=%s fails:\n' "$1" "$path" >>"$report"
        cat "$tmp/out" >>"$report"
    fi
}

# gates prints the targets of make that check the file $path, by its kind: none for a file of no
# kind a gate checks. A file without an extension a gate knows may be a shell script by its
# shebang, as tools/files finds it; where tools/files fails, the shell gate reports why.
gates() {
    case $path in
    *.md) echo vale lychee rumdl sizecheck ;;
    .github/workflows/*.yml | .github/workflows/*.yaml)
        echo yamllint ansible-lint actionlint zizmor sizecheck
        ;;
    *.yml | *.yaml) echo yamllint ansible-lint sizecheck ;;
    *.py) echo ruff sizecheck ;;
    *.j2) echo sizecheck ;;
    *)
        if ! tools/files sh "$path" </dev/null >"$tmp/out" 2>&1; then
            echo shell
        elif [ -s "$tmp/out" ]; then
            echo shell sizecheck
        fi
        ;;
    esac
}

# remember adds the work tree $1 to the record of the session that $input names, which
# lint-branch.sh lints at Stop and then clears.
remember() {
    record=$(record_path "$repo" "$(printf '%s\n' "$input" | field session_id)") || return 0
    mkdir -p -- "${record%/*}" || return 0
    grep -qxF -- "$1" "$record" 2>/dev/null || printf '%s\n' "$1" >>"$record"
}

main() {
    # stdin is read whole first, as the hook reads two of its fields.
    input=$(cat)
    file=$(printf '%s\n' "$input" | field file_path)
    # A path with an escape in its JSON, a backslash or a quote, is none of the repository's.
    case $file in
    *\\*) return 0 ;;
    esac
    [ -f "$file" ] || return 0
    dir=$(dirname -- "$file")
    repo=$(common "$hooks")
    if [ -z "$repo" ] || [ "$(common "$dir")" != "$repo" ]; then
        return 0
    fi
    # The file's own work tree, which may be a linked one, and the file's path in it. A branch
    # from before the gates has no tools/files.
    cd -- "$dir" || return 0
    path=$(git rev-parse --show-prefix)$(basename -- "$file")
    top=$(git rev-parse --show-toplevel)
    cd -- "$top" || return 0
    if git check-ignore -q -- "$path" || [ ! -f tools/files ]; then
        return 0
    fi
    # The path goes into make's command line, and the tree into the record and the JSON.
    if ! plain "$top/$path"; then
        notice "lint-file.sh: no gate checks the file, as its path holds a character outside \
[A-Za-z0-9._/@+-] or a part starting with -."
        return 0
    fi
    remember "$top"
    if ! vouched; then
        notice "$(unvouched "$top")"
        return 0
    fi

    tmp=$(mktemp -d) || return 0
    trap 'rm -rf "$tmp"' EXIT
    report=$tmp/report
    : >"$report"
    for target in $(gates); do
        gate "$target"
    done
    if [ -s "$report" ]; then
        cat "$report" >&2
        return 2
    fi
}

main "$@"
