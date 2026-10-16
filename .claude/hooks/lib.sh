# shellcheck shell=sh
# The functions of Claude Code's hooks here, sourced by lint-file.sh and lint-branch.sh. Sourcing it
# defines functions and base, and changes nothing else.

# base is the full ref of origin/main. git reads a tag or a branch named origin/main before the
# remote's, and a fetch from a fork may bring such a tag.
base=refs/remotes/origin/main

# field prints the string field $1 of the hook's JSON on stdin, escapes as they are. sed reads it,
# as jq may be missing. A quote in a JSON string is escaped, so only a key matches.
field() {
    tr '\n' ' ' | sed -E -n "s/.*\"$1\"[[:space:]]*:[[:space:]]*\"(([^\"\\\\]|\\\\.)*)\".*/\1/p"
}

# common prints the git common directory of the directory $1, which every work tree of a
# repository shares.
common() {
    git -C "$1" rev-parse --path-format=absolute --git-common-dir 2>/dev/null
}

# plain succeeds where the path $1 holds only characters of [A-Za-z0-9._/@+-] and no part of it
# starts with -. make pastes a path into its recipes unquoted, and expands a $ in it, and a tool
# takes a part starting with - as an option. A path that passes is safe in the hooks' JSON too.
plain() {
    case $1 in
    '' | *[!A-Za-z0-9._/@+-]* | -* | */-*) return 1 ;;
    esac
}

# vouched succeeds where the maintainer vouches for the code of the work tree in the current
# directory, which the gates run. Each commit from origin/main to HEAD takes a good signature,
# %G? of G, by the key that gpg.ssh.allowedSignersFile lists for its author, %ae, which no
# .mailmap changes. A rebase or cherry-pick here signs another's commit too, but keeps its author.
# A git config key would not do: worktrees share their clone's config, so it would vouch for an
# outside pull request fetched into a worktree too.
#
# The index matches HEAD, with no conflict, as a merge, a cherry-pick or a checkout of a path
# stages another's changes. Those may hold code in any file, such as a lookup in the playbook's
# YAML, which ansible-lint's syntax check runs.
#
# The gates' code matches HEAD in the work tree too, with no file of it untracked, as git apply
# stages nothing. Its pathspecs: make's makefiles and tools/, the pins of the Python tools and
# collections, the settings of ansible and ansible-lint, the inventory, and .cache/. So is any
# Python, bytecode or extension module: ansible-lint loads them, and python3 -m venv imports them
# from the tree's root. .gitignore is one too, as ls-files leaves out what it ignores. An unstaged
# change to another file, such as the YAML, goes unseen: the maintainer's edits look the same.
vouched() {
    signs=$(git log --no-show-signature --format='%G? %GS %ae' "$base..HEAD" 2>/dev/null) ||
        return 1
    printf '%s\n' "$signs" | awk 'NF && !(NF == 3 && $1 == "G" && $2 == $3) { exit 1 }' ||
        return 1
    git diff --cached --quiet HEAD -- 2>/dev/null || return 1
    set -- GNUmakefile makefile Makefile tools .github/lint-requirements.in \
        .github/lint-requirements.txt requirements.yml ansible.cfg '.ansible-lint*' .config \
        inventory.ini '*.py' '*.pyc' '*.so' .cache .gitignore
    git diff --quiet HEAD -- "$@" 2>/dev/null || return 1
    untracked=$(git ls-files --others --exclude-standard -- "$@" 2>/dev/null) &&
        [ -z "$untracked" ]
}

# unvouched prints the line that tells the user why no gate runs in the work tree $1.
unvouched() {
    printf '%s: no gate runs in %s, %s %s\n' "${0##*/}" "$1" \
        "as a commit since origin/main lacks its author's good signature, a change is staged," \
        "the gates' code differs from HEAD, or origin/main is missing."
}

# record_path prints the file in the git common directory $1 that holds the work trees the
# session $2 wrote, a line each. It fails where $2 is not of a session id's characters.
record_path() {
    case $2 in
    '' | *[!A-Za-z0-9-]*) return 1 ;;
    esac
    printf '%s/claude-lint/%s\n' "$1" "$2"
}

# notice shows the user the line $1, which holds no quote or backslash, in the hook's JSON output,
# and lets Claude go on.
notice() {
    printf '{"systemMessage": "%s"}\n' "$1"
}
