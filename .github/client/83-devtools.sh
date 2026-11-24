#!/bin/bash
# mise bootstrap (28-bootstrap.sh) installed macos/roles/devtools' tools. Go comes by the lock under
# the user's mise data directory, which /usr/local/go links to. The formulae are on a login
# shell's PATH by /etc/paths.d/ws. The bootstrap seeded gpg-agent.conf, in a ~/.gnupg at 0700.

set -eo pipefail

fail=0
[ "$(readlink /usr/local/go)" = "$HOME/.local/share/mise/installs/go/1.27.3" ] ||
    {
        echo "::error::/usr/local/go does not link to mise's Go 1.27.3"
        fail=1
    }
grep -qxF /usr/local/go/bin /etc/paths.d/ws ||
    {
        echo '::error::/etc/paths.d/ws does not name /usr/local/go/bin'
        fail=1
    }
goversion=$(env -i HOME="$HOME" zsh -lc 'go version') || goversion=
case "$goversion" in
'go version go1.27.3 darwin/arm64') ;;
*)
    echo "::error::a login shell's go prints '$goversion', not go1.27.3 darwin/arm64"
    fail=1
    ;;
esac
for tool in gpg pinentry-mac ykman rg tokei tree pwgen; do
    env -i HOME="$HOME" zsh -lc "command -v $tool" | grep -qx "/opt/homebrew/bin/$tool" ||
        {
            echo "::error::a login shell's $tool is not Homebrew's in /opt/homebrew/bin"
            fail=1
        }
done
[ "$(stat -f '%Lp' ~/.gnupg)" = 700 ] ||
    {
        echo '::error::~/.gnupg is not at 0700'
        fail=1
    }
grep -qxF 'pinentry-program /opt/homebrew/bin/pinentry-mac' ~/.gnupg/gpg-agent.conf ||
    {
        echo '::error::gpg-agent.conf does not name pinentry-mac'
        fail=1
    }
exit "$fail"
