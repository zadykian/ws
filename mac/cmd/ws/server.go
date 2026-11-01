package main

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// osRelease is where the system names itself.
const osRelease = "/etc/os-release"

// serverRefusal returns why ws server apply does not run here, or "" where it does: as root on
// Ubuntu 26.04, the one system that the playbook is written for. osRelease is /etc/os-release.
func serverRefusal(euid int, osRelease []byte) string {
	if euid != 0 {
		return "run it as root"
	}
	var id, version string
	sc := bufio.NewScanner(bytes.NewReader(osRelease))
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "=")
		if !ok {
			continue
		}
		v = strings.Trim(v, `"'`)
		switch k {
		case "ID":
			id = v
		case "VERSION_ID":
			version = v
		}
	}
	if id != "ubuntu" || version != "26.04" {
		return fmt.Sprintf("the playbook is for Ubuntu 26.04, not %s %s", id, version)
	}
	return ""
}

// serverApplyArgs returns the arguments for ansible-playbook: those after "--". Anything else
// before it is not accepted.
func serverApplyArgs(args []string) ([]string, bool) {
	if len(args) == 0 {
		return nil, true
	}
	if args[0] != "--" {
		return nil, false
	}
	return args[1:], true
}
