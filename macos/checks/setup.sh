#!/bin/bash
# The real Mac's checks of its first run, its login items and its shell: macos/checks/README.md.
# Run as the Mac's user, not root, as
#
#   setup.sh [CHECK...]
#
# CHECK is a trial's number on the page. 1 is the first run's end state, 5 the login items and 10
# the shell. Without one, it runs each. It prints a line a check, ok or FAIL, and exits 1 where any
# fails. It changes nothing: the first run and the second are by hand, as the page says.
set -uo pipefail

dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=macos/checks/lib.sh
. "$dir/lib.sh"

# formulae prints the name of each formula that a role names, a line each.
formulae() { sed -n 's/^"brew:\([^"]*\)".*/\1/p' "$tree"/macos/roles/*/mise.macos.toml; }

# missing_formulae prints the formulae that stand nowhere in Homebrew's prefix, as one line.
missing_formulae() {
    local name gone=
    for name in $(formulae); do
        [ -e "$brew/opt/$name" ] || gone="$gone $name"
    done
    [ -z "$gone" ] || {
        echo "missing:$gone"
        return 1
    }
}

# missing_apps prints the apps of the Dock and the login items that are not there.
missing_apps() {
    local app gone=
    while IFS= read -r app; do
        [ -d "$app" ] || gone="$gone $app;"
    done < <(pinned_apps)
    [ -z "$gone" ] || {
        echo "missing:$gone"
        return 1
    }
}

# pinned_apps prints each app that the Dock or the login items name, once.
pinned_apps() { printf '%s\n%s\n' "$(dock_apps)" "$(login_apps)" | sort -u; }

# dock_order succeeds where the Dock's apps are the pinned list, in its order.
dock_order() {
    local want got
    want=$(dock_apps | sed 's|.*/||')
    got=$(defaults read com.apple.dock persistent-apps | sed -n 's|.*"_CFURLString" = "file://.*/\([^/]*\.app\)/*";|\1|p' | sed 's/%20/ /g')
    [ "$want" = "$got" ] || {
        echo "the Dock holds: $(printf '%s' "$got" | tr '\n' ',')"
        return 1
    }
}

# go_of_mise succeeds where a login shell's go, asked from the home directory, is the lock's Go.
go_of_mise() {
    local want out
    want=$(sed -n 's/^go = "\([^"]*\)".*/\1/p' "$tree/macos/roles/devtools/mise.macos.toml")
    out=$(cd "$HOME" && zsh -lc 'go version') || return 1
    case $out in
    "go version go$want darwin/arm64") ;;
    *)
        echo "go prints '$out', not go$want"
        return 1
        ;;
    esac
}

check1() {
    check "1 setup's plan exits 0, in both envs" plan_zero
    check "1 mise's status finds nothing missing" status_zero
    check "1 each formula stands in Homebrew's prefix" missing_formulae
    check "1 each app of the Dock and the login items is installed" missing_apps
    check "1 the Dock holds its apps in order" dock_order
    check "1 a login shell's go is the lock's, from the home directory" go_of_mise
    check "1 ykman runs" ykman --version
    check "1 colima, docker and its plugins answer" docker_cli
}

docker_cli() {
    colima version >/dev/null && docker --version >/dev/null &&
        docker compose version >/dev/null && docker buildx version >/dev/null
}

# btm_count APP prints the records of sfltool's dump that name APP as a login item. The URL is a
# file URL, which writes a space as %20.
btm_count() {
    awk -v app="/$1/" '
        /^ *#[0-9]+:/ { if (url != "" && login) n++; url = ""; login = 0 }
        /^ *URL:/ { u = $0; gsub(/%20/, " ", u); if (index(u, app)) url = u }
        /^ *Type:.*login item/ { login = 1 }
        END { if (url != "" && login) n++; print n + 0 }
    ' <<<"$2"
}

check5() {
    local dump app name n
    dump=$(sudo sfltool dumpbtm 2>&1) || {
        fail "5 sfltool dumpbtm answers" "$(printf '%s' "$dump" | tail -n 1)"
        return
    }
    check "5 mise's status holds the login items" status_zero
    while IFS= read -r app; do
        name=${app##*/}
        n=$(btm_count "$name" "$dump")
        if [ "$n" -eq 1 ]; then pass "5 $name opens once at login"; else
            fail "5 $name opens once at login" "$n records name it as a login item"
        fi
    done < <(login_apps)
    check "5 Secretive's agent runs" pgrep -x SecretAgent
    check "5 Logi's agent runs" pgrep -f logioptionsplus_agent
    check "5 Rectangle Pro runs" pgrep -x "Rectangle Pro"
    check "5 Stats runs" pgrep -x Stats
    info "login items in the dump: $(grep -c 'Type:.*login item' <<<"$dump")"
}

check10() {
    local out
    out=$(zsh -ic 'print -r "plugins=${plugins[*]}"
        print -r "autosuggestions=${+functions[_zsh_autosuggest_start]}"
        print -r "highlighting=${+functions[_zsh_highlight]}"
        zstyle -s ":omz:update" mode m && print -r "update=$m"' </dev/null 2>&1)
    check "10 a new shell loads the three plugins" grep -qx 'plugins=git zsh-autosuggestions zsh-syntax-highlighting' <<<"$out"
    check "10 zsh-autosuggestions is loaded" grep -qx 'autosuggestions=1' <<<"$out"
    check "10 zsh-syntax-highlighting is loaded" grep -qx 'highlighting=1' <<<"$out"
    check "10 oh-my-zsh's update is off" grep -qx 'update=disabled' <<<"$out"
    if grep -qi 'insecure directories\|update oh my zsh' <<<"$out"; then
        fail "10 the shell asks nothing" "$(grep -i 'insecure directories\|update oh my zsh' <<<"$out" | head -n 1)"
    else
        pass "10 the shell asks nothing"
    fi
}

main() {
    local arg all=(1 5 10)
    [ $# -gt 0 ] || set -- "${all[@]}"
    for arg in "$@"; do
        case $arg in
        1) check1 ;;
        5) check5 ;;
        10) check10 ;;
        *)
            echo "setup.sh: no check $arg here" >&2
            exit 2
            ;;
        esac
    done
    finish
}

# checks-test.sh sources the file for its functions.
[ "${BASH_SOURCE[0]}" != "$0" ] || main "$@"
