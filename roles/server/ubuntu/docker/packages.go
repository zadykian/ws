package docker

import "github.com/zadykian/ws/roles/server/ubuntu/values"

// rootlessPackages are what ws_user's daemon needs beyond the system's. apparmor is named for its
// rootlesskit profile, which Ubuntu's kernel asks for before it grants an unprivileged user
// namespace: Docker's packages only recommend it.
var rootlessPackages = []string{"docker-ce-rootless-extras", "uidmap", "slirp4netns", "apparmor"}

// packages lists what the Docker packages step installs.
func packages(v values.Values) []string {
	names := append(append([]string{}, v.Docker.Packages...), "iproute2")
	if v.User != "root" {
		names = append(names, rootlessPackages...)
	}
	return names
}
