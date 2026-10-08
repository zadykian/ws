#!/bin/sh
# Claude Code's PostToolUse hook for Edit and Write (.claude/settings.json): runs make lint's gates
# on the file just written, by its kind. A finding goes to stderr with status 2, which Claude sees;
# a file outside this repository, or one git ignores, such as a plan, is passed over.

set -u

# file_path prints tool_input.file_path of the hook's JSON on stdin. sed reads it, as jq may be
# missing; a quote in a JSON string is escaped, so only a key matches.
file_path() {
    tr '\n' ' ' | sed -E -n 's/.*"file_path"[[:space:]]*:[[:space:]]*"(([^"\\]|\\.)*)".*/\1/p'
}

# common prints the git common directory of the directory $1, which every work tree of a
# repository shares.
common() {
    git -C "$1" rev-parse --path-format=absolute --git-common-dir 2>/dev/null
}

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

main() {
    file=$(file_path)
    # A path with an escape in its JSON, a backslash or a quote, is none of the repository's.
    case $file in
    *\\*) return 0 ;;
    esac
    [ -f "$file" ] || return 0
    dir=$(dirname -- "$file")
    repo=$(common "$(dirname -- "$0")")
    if [ -z "$repo" ] || [ "$(common "$dir")" != "$repo" ]; then
        return 0
    fi
    # The file's own work tree, which may be a linked one, and the file's path in it. A branch
    # from before the gates has no tools/files.
    cd -- "$dir" || return 0
    path=$(git rev-parse --show-prefix)$(basename -- "$file")
    cd -- "$(git rev-parse --show-toplevel)" || return 0
    if git check-ignore -q -- "$path" || [ ! -f tools/files ]; then
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
