#!/bin/bash
# The real Mac's checks of its first run, its login items and its shell: trials 1, 5 and 10.
# Run as the Mac's user, not root, as `setup.sh [CHECK...]`, a trial's number each or none for
# all three. It changes nothing. The page is macos/checks/README.md.
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
    [ -n "$want" ] || {
        echo "devtools' file names no Go version"
        return 1
    }
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

# btm_logins DUMP prints the URL of each login item in sfltool's dump, a line each. A file URL
# writes a space as %20.
btm_logins() {
    awk '
        function item() { if (url != "" && login) print url; url = ""; login = 0 }
        /^ *#[0-9]+:/ { item() }
        /^ *URL:/ { url = $0; sub(/^ *URL: */, "", url); sub(/ *$/, "", url); gsub(/%20/, " ", url) }
        /^ *Type:.*login item/ { login = 1 }
        END { item() }
    ' <<<"$1"
}

# btm_count APP DUMP prints the login items whose URL is APP's path whole, so that a bundle inside
# APP's own counts apart.
btm_count() { btm_logins "$2" | grep -cxF "file://$1/"; }

# login_status succeeds where mise's status finds each login item set, and none undeclared. An
# undeclared one is an app's own item of the shared list, which exclusive takes away at each run.
login_status() {
    local out rows
    out=$("$mise" -C "$tree/macos" -E ws,apps bootstrap status 2>&1) || {
        printf '%s\n' "$out" | tail -n 1
        return 1
    }
    rows=$(grep '^login items' <<<"$out")
    [ -n "$rows" ] || {
        echo "mise's status lists no login item"
        return 1
    }
    rows=$(grep -v ' set$' <<<"$rows")
    [ -z "$rows" ] || {
        printf '%s\n' "$rows" | tr -s ' ' | tr '\n' ';'
        return 1
    }
}

check5() {
    local dump app name n others
    dump=$(sudo sfltool dumpbtm 2>&1) || {
        fail "5 sfltool dumpbtm answers" "$(printf '%s' "$dump" | tail -n 1)"
        return
    }
    check "5 mise's status finds each login item set, and none undeclared" login_status
    while IFS= read -r app; do
        name=${app##*/}
        n=$(btm_count "$app" "$dump")
        if [ "$n" -eq 1 ]; then pass "5 $name opens once at login"; else
            fail "5 $name opens once at login" "$n records name it as a login item"
        fi
    done < <(login_apps)
    check "5 seward's agent runs" pgrep -x seward
    check "5 Logi's agent runs" pgrep -f logioptionsplus_agent
    check "5 Rectangle Pro runs" pgrep -x "Rectangle Pro"
    check "5 Stats runs" pgrep -x Stats
    others=$(btm_logins "$dump" | grep -vxF -f <(login_apps | sed 's|.*|file://&/|') | sort -u | tr '\n' ';')
    info "5 the dump's login items outside the role: ${others:-none}"
}

check10() {
    # What oh-my-zsh prints at its pinned commit: compfix's warning, and the update's prompt.
    local out asks='insecure (completion-dependent )?directories|would you like to update|time to update'
    out=$(zsh -ic 'print -r "plugins=${plugins[*]}"
        print -r "autosuggestions=${+functions[_zsh_autosuggest_start]}"
        print -r "highlighting=${+functions[_zsh_highlight]}"
        zstyle -s ":omz:update" mode m && print -r "update=$m"' </dev/null 2>&1)
    check "10 a new shell loads the three plugins" grep -qx 'plugins=git zsh-autosuggestions zsh-syntax-highlighting' <<<"$out"
    check "10 zsh-autosuggestions is loaded" grep -qx 'autosuggestions=1' <<<"$out"
    check "10 zsh-syntax-highlighting is loaded" grep -qx 'highlighting=1' <<<"$out"
    check "10 oh-my-zsh's update is off" grep -qx 'update=disabled' <<<"$out"
    if grep -Eqi "$asks" <<<"$out"; then
        fail "10 the shell asks nothing" "$(grep -Ei "$asks" <<<"$out" | head -n 1)"
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
