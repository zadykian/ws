package secrets

import (
	"context"
	"fmt"
	"path"
)

// Hostname is the hostname of the server that the Mac names name, which serves as its profile. A
// label of fnox.toml gives it, where that label alone holds name's host key in home's known_hosts,
// as Drop finds it. Else ask reads it from the server, as add does by ssh.
func (c *Clone) Hostname(ctx context.Context, home, name string, ask func(context.Context) (string, error)) (string, error) {
	conf, found, err := ReadConfig(c.M, path.Join(c.Dir, SharedFile), SharedProvider)
	if err != nil {
		return "", err
	}
	if !found {
		return "", errNotSetUp
	}
	key, err := HostKey(ctx, c.M, home, name)
	if err != nil {
		return "", err
	}
	var labels []string
	for label, recipient := range conf.Recipients {
		if key != "" && recipient == key {
			labels = append(labels, label)
		}
	}
	hostname := ""
	if len(labels) == 1 {
		hostname = labels[0]
	} else if hostname, err = ask(ctx); err != nil {
		return "", err
	}
	if err := CheckHostname(hostname); err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return hostname, nil
}
