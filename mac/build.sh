#!/bin/sh
# ws_build builds ws, the Go CLI in the clone's mac/, which bootstrap.sh and setup.sh each source
# from the clone. It is POSIX sh, and has no functions of lib.sh, as bootstrap.sh runs before it.

# ws_build CLONE FILE builds ws from CLONE into FILE, with the Go that tools/run downloads and
# checks into the clone's .cache, without cgo. It leaves out the commit's stamp, so that a new
# commit leaves the bytes as they were and setup.sh restarts the daemon only where its code
# changed. It writes FILE.new and moves it over FILE, so that a daemon never runs a half-written
# binary. On failure it prints why to stderr and returns 1.
ws_build() {
    ws_go=$(cd "$1" && sh tools/run -path go) || {
        echo "cannot get the pinned Go" >&2
        return 1
    }
    mkdir -p "${2%/*}" || {
        echo "cannot make ${2%/*}" >&2
        return 1
    }
    (
        cd "$1/mac" &&
            GOPATH=$1/.cache/go GOCACHE=$1/.cache/go-build GOTOOLCHAIN=local GOWORK=off CGO_ENABLED=0 \
                "$ws_go" build -buildvcs=false -o "$2.new" ./cmd/ws
    ) || {
        echo "cannot build ws" >&2
        return 1
    }
    mv -f "$2.new" "$2" || {
        echo "cannot install $2" >&2
        return 1
    }
}
