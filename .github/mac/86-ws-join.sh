#!/bin/bash
# ws mac apply writes ws-join into ~/.local/bin: mac/ws-join whole, mode 755. A stand-in ssh
# exits with the statuses given, in turn, and logs a line where given. A 255 then a 0 make two
# calls, the second with --existing. A 1 makes one, and so does a 255 whose log says that ws
# refused the login.

set -eo pipefail

fail=0
join=~/.local/bin/ws-join
cmp -s "$join" mac/ws-join && [ -x "$join" ] ||
    {
        echo '::error::~/.local/bin/ws-join is not mac/ws-join, mode 755'
        fail=1
    }
bin=$RUNNER_TEMP/join-bin
mkdir -p "$bin"
cat >"$bin/ssh" <<'SSH'
#!/bin/sh
echo "$*" >>"$CALLS"
n=$(($(wc -l <"$CALLS")))
while [ "$#" -gt 1 ]; do
    [ "$1" = -E ] && [ -n "$LOGGED" ] && echo "$LOGGED" >>"$2"
    shift
done
exit "$(echo "$STATUSES" | cut -d ' ' -f "$n")"
SSH
chmod 755 "$bin/ssh"

# join STATUSES LOGGED CALLS STATUS runs ws-join and checks the number of calls and its status.
join() {
    rm -f "$RUNNER_TEMP/calls"
    status=0
    CALLS=$RUNNER_TEMP/calls STATUSES=$1 LOGGED=$2 PATH=$bin:$PATH "$join" -n s1 -s a </dev/null ||
        status=$?
    cat "$RUNNER_TEMP/calls"
    [ "$(($(wc -l <"$RUNNER_TEMP/calls")))" = "$3" ] && [ "$status" = "$4" ] ||
        {
            echo "::error::ws-join on $1, '$2': status $status; want $3 calls and status $4"
            fail=1
        }
}
join '255 0' 'client_loop: send disconnect: Broken pipe' 2 0
grep -q 'cld join --existing -n s1 -s a' "$RUNNER_TEMP/calls" ||
    {
        echo '::error::the second try has no --existing'
        fail=1
    }
join 1 '' 1 1
join '255 0' 'ada@10.99.0.1: Permission denied (publickey).' 1 255
exit "$fail"
