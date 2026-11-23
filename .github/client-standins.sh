#!/bin/sh
# The stand-ins of the client job (.github/workflows/client.yml), made before its first mise bootstrap
# (28-bootstrap.sh). They spare each run the downloads, which client-cld.yml makes for real, so that
# the job's many runs that must change nothing touch no network:
#
#   sh .github/client-standins.sh
#
# - wireguard-tools from Homebrew where missing, whose wg and wireguard-go make the stand-in ws.
#   The daemon needs neither, so this is the one download, a small formula, that the jobs make;
# - ~/.local/bin/cld, whose version names no release, so that cld's unit neither installs nor
#   updates it, as a cld built from a clone. It lacks restore tabs, so ssh's hosts get no
#   LocalCommand.

set -eu

if [ -e ~/.local/bin/cld ]; then
    echo '::error::the runner has ~/.local/bin/cld already'
    exit 1
fi
brew list --formula wireguard-tools >/dev/null 2>&1 || brew install wireguard-tools
mkdir -p ~/.local/bin
cat >~/.local/bin/cld <<'EOS'
#!/bin/sh
# A stand-in cld: it answers --version with no release's, and fails at anything else.
[ "$*" = --version ] && echo 'cld stand-in'
EOS
chmod 755 ~/.local/bin/cld
