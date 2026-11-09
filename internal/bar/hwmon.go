package bar

import (
	"io/fs"
	"strconv"
	"strings"
)

// hottest returns the highest temperature, in whole °C, of the channels of every hwmon device
// named device whose label starts with label. An empty label takes every channel, as spd5118's
// have none. A machine without such a device, as a VM, has no reading.
func hottest(fsys fs.FS, device, label string) (int, bool) {
	dirs, _ := fs.Glob(fsys, "sys/class/hwmon/hwmon*") // the pattern is ours, so it holds no syntax error
	best, found := 0, false
	for _, dir := range dirs {
		if name, err := fs.ReadFile(fsys, dir+"/name"); err != nil || strings.TrimSpace(string(name)) != device {
			continue
		}
		inputs, _ := fs.Glob(fsys, dir+"/temp*_input")
		for _, input := range inputs {
			if label != "" {
				text, err := fs.ReadFile(fsys, strings.TrimSuffix(input, "_input")+"_label")
				if err != nil || !strings.HasPrefix(strings.TrimSpace(string(text)), label) {
					continue
				}
			}
			text, err := fs.ReadFile(fsys, input)
			if err != nil {
				continue
			}
			milli, err := strconv.Atoi(strings.TrimSpace(string(text)))
			if err != nil {
				continue
			}
			if c := milli / 1000; !found || c > best {
				best, found = c, true
			}
		}
	}
	return best, found
}
