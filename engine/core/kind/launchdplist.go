package kind

import (
	"bytes"
	"encoding/xml"

	"github.com/zadykian/ws/engine/core/resource/file"
)

// plistText is a daemon's plist. launchd runs it at load and from boot, before any login, starts it
// again 10 s after it exits, and appends its output to its log, where it has one.
func plistText(label string, d launchdDaemon) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<!-- ` + file.Mark + ` -->
<plist version="1.0">
<dict>
`)
	str := func(indent, s string) {
		b.WriteString(indent + "<string>")
		_ = xml.EscapeText(&b, []byte(s))
		b.WriteString("</string>\n")
	}
	key := func(k string) { b.WriteString("\t<key>" + k + "</key>\n") }
	key("Label")
	str("\t", label)
	key("ProgramArguments")
	b.WriteString("\t<array>\n")
	for _, a := range d.Arguments {
		str("\t\t", a)
	}
	b.WriteString("\t</array>\n")
	key("RunAtLoad")
	b.WriteString("\t<true/>\n")
	key("KeepAlive")
	b.WriteString("\t<true/>\n")
	key("ThrottleInterval")
	b.WriteString("\t<integer>10</integer>\n")
	if d.Log != "" {
		key("StandardOutPath")
		str("\t", d.Log)
		key("StandardErrorPath")
		str("\t", d.Log)
	}
	b.WriteString("</dict>\n</plist>\n")
	return b.Bytes()
}
