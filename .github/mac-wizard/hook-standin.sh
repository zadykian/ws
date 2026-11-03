#!/bin/sh
# The stand-in for the released command of WS_HOOK's repository, as the stand-in gh on the Mac
# downloads it and ws runs it, in ws's WS_STANDIN, which ws's sshd sets. Each run logs a line:
#
# - apply.log holds tty where stdin is a terminal, else the SHA-256 of what the command read there,
#   and apply.status its status;
# - check.log holds the run with --check instead, and hook.args the arguments of each run.

s=$WS_STANDIN
echo "$*" >>"$s/hook.args"
[ "${1-}" = apply ] || exit 2
log=apply.log
case " $* " in *' --check '*) log=check.log ;; esac
if [ -t 0 ]; then echo tty; else shasum -a 256 | cut -d ' ' -f 1; fi >>"$s/$log"
exit "$(cat "$s/apply.status" 2>/dev/null || echo 0)"
