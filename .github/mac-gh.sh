# shellcheck shell=sh
# A stand-in gh goes in DIR/bin, logged in while DIR/login is there. It logs each call, and lists and
# adds the account's signing keys in DIR/keys, for ws client macos apply's commands alone. With
# DIR/error, it fails with that text; with DIR/lost, an add keeps no key.
#
# Run it as sh .github/mac-gh.sh DIR, and put DIR/bin on PATH.
set -eu
dir=${1:?the directory of the stand-in}
mkdir -p "$dir/bin"
: >"$dir/keys"
touch "$dir/login"
cat >"$dir/bin/gh" <<'STANDIN'
#!/bin/sh
state=${0%/bin/*}
echo "$*" >>"$state/gh.log"
if [ ! -f "$state/login" ]; then
    echo 'To get started with GitHub CLI, please run:  gh auth login' >&2
    exit 4
fi
[ ! -f "$state/error" ] || { cat "$state/error" >&2 && exit 1; }
case $* in
'api --paginate user/ssh_signing_keys --jq .[].key') cat "$state/keys" ;;
'ssh-key add '*) [ -f "$state/lost" ] || awk '{ print $1, $2 }' "$3" >>"$state/keys" ;;
*) echo "gh: the stand-in takes no $*" >&2 && exit 2 ;;
esac
STANDIN
chmod 755 "$dir/bin/gh"
