#!/bin/bash
# The install job's cases of the autoinstall file. Run it from the repository's root, where
# cloud-init and python3 are installed: CI runs it in an Ubuntu 26.04 container, whose cloud-init
# is the installer's. It changes nothing outside a temporary directory.
#
# autoinstall.sh fills a release's blanks and refuses a bad tag or SUM. The file it prints must
# keep the maintainer's blanks alone. Its two layers must pass `cloud-init schema`. They are the
# whole file, as NoCloud reads it, and its user-data, which the installed machine's cloud-init
# reads. Each command of its late-commands must parse as shell. The one that fetches install.sh
# runs against a curl that serves the file only to a request that follows redirects, as GitHub's
# release address needs.

set -euo pipefail

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
tag=v0.13.0-rc.1
# The file that the stub curl below serves, and its SHA-256, which the filled file checks.
served=': the install.sh of a release'
sum=$(printf '%s\n' "$served" | sha256sum | cut -d ' ' -f 1)

fail() {
    echo "::error::$*"
    exit 1
}

.github/autoinstall.sh "$tag" "$sum" >"$tmp/user-data"

[ "$(head -n 1 "$tmp/user-data")" = '#cloud-config' ] || fail "the filled file does not open with #cloud-config"
[ "$(sed -n 2p "$tmp/user-data")" = '---' ] || fail "the filled file lost its document start"
! grep -qE '\<(VERSION|SHA256)\>' "$tmp/user-data" || fail "the filled file keeps a blank of the release's"
grep -qF "releases/download/$tag/install.sh" "$tmp/user-data" || fail "the file names no download of $tag"
grep -qF "$sum  " "$tmp/user-data" || fail "the file checks no SHA-256 of install.sh"
[ "$(grep -cF "WS_VERSION=$tag " "$tmp/user-data")" = 1 ] || fail "install.sh does not run once with WS_VERSION=$tag"
grep -qF -e "--version, $tag]" "$tmp/user-data" || fail "the first boot runs no ws update --version $tag"
# Each blank at the key it fills: the header comment names them too, so a bare word would pass.
for blank in 'hostname: HOSTNAME' 'password: HASH' 'content: VALUES'; do
    grep -qE "^ +$blank\$" "$tmp/user-data" || fail "the maintainer's blank '$blank' is gone"
done

# The refusals: a tag that is no release's, and a SUM that is no SHA-256.
for args in "" "$tag" "0.13.0 $sum" "v0.13 $sum" "v0.13.0;x $sum" "$tag ${sum:1}" "$tag ${sum^^}"; do
    # shellcheck disable=SC2086 # the words of each case are its arguments
    if .github/autoinstall.sh $args >/dev/null 2>&1; then
        fail "autoinstall.sh took the arguments: $args"
    fi
done

cloud-init schema --config-file "$tmp/user-data" || fail "cloud-init refuses the file"

# The file's user-data and its late-commands, each as the file of its own.
python3 - "$tmp" <<'PY'
import sys
import yaml

tmp = sys.argv[1]
doc = yaml.safe_load(open(f"{tmp}/user-data"))["autoinstall"]
with open(f"{tmp}/machine-data", "w") as out:
    out.write("#cloud-config\n" + yaml.safe_dump(doc["user-data"]))
for n, argv in enumerate(doc["late-commands"]):
    # An argv of curtin in-target, whose command after -- is sh -ec SCRIPT.
    with open(f"{tmp}/late-{n}.sh", "w") as out:
        out.write(argv[-1] + "\n")
PY
cloud-init schema --config-file "$tmp/machine-data" || fail "cloud-init refuses the file's user-data"
for script in "$tmp"/late-*.sh; do
    sh -n "$script" || fail "$(basename "$script") of late-commands does not parse"
done

# The late command that fetches install.sh runs against a stub curl. A request that does not
# follow redirects gets an empty file and a success, as GitHub's release address answers it.
mkdir "$tmp/bin"
cat >"$tmp/bin/curl" <<STUB
#!/bin/sh
follow=
while [ \$# -gt 0 ]; do
    case \$1 in
    --location | -[a-zA-Z]*L*) follow=1 ;;
    -o) out=\$2; shift ;;
    esac
    shift
done
if [ -n "\$follow" ]; then
    echo '$served' >"\$out"
else
    : >"\$out"
fi
STUB
chmod +x "$tmp/bin/curl"
fetch=$(grep -lF 'releases/download' "$tmp"/late-*.sh)
[ "$(printf '%s\n' "$fetch" | wc -l)" = 1 ] || fail "not one late command fetches install.sh"
PATH="$tmp/bin:$PATH" sh -e "$fetch" >/dev/null ||
    fail "the late command that fetches install.sh fails: does its curl follow redirects?"
echo "the autoinstall file passes"
