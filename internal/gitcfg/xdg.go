package gitcfg

import "fmt"

// XDGFile is git's XDG config file, which git reads before ~/.gitconfig: xdg is XDG_CONFIG_HOME,
// empty where unset, and home the account's home.
func XDGFile(xdg, home string) string {
	if xdg == "" {
		xdg = home + "/.config"
	}
	return xdg + "/git/config"
}

// XDGNote is the warning that settings in git's XDG file lose to the include's file, as each side
// words it. file is the XDG file, own the settings of Own and include the file that wins.
func XDGNote(file, own, include string) string {
	return fmt.Sprintf("%s's settings of %s come before the include in ~/.gitconfig, so %s's win over them",
		file, own, include)
}
