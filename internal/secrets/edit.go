package secrets

import (
	"fmt"
	"slices"
)

// The kinds of an entry, as ws mac secrets names them.
const (
	KindToken    = "token"
	KindEnv      = "env"
	KindGit      = "git"
	KindFile     = "file"
	KindPassword = "password"
)

// Set puts value under the kind and the name: a token, an env variable or a password. A git
// entry and a file have their own forms, SetGit and SetFile. The error names the kind and the
// name, never the value; Decode checks the name of an env variable when the file is written.
func (s *Secrets) Set(kind, name, value string) error {
	switch {
	case kind == KindToken && name == "github":
		s.Token.GitHub = value
	case kind == KindToken && name == "azure_devops":
		s.Token.AzureDevOps = value
	case kind == KindPassword && name == "vpn_p12":
		s.Password.VPNP12 = value
	case kind == KindEnv:
		if s.Env == nil {
			s.Env = map[string]string{}
		}
		s.Env[name] = value
	default:
		return unknown(kind, name)
	}
	return nil
}

// SetGit replaces the credential of the server at url.
func (s *Secrets) SetGit(url string, g Git) {
	if s.Git == nil {
		s.Git = map[string]Git{}
	}
	s.Git[url] = g
}

// SetFile puts the bytes under the name of a profile file.
func (s *Secrets) SetFile(name string, data []byte) error {
	switch name {
	case "vpn_p12":
		s.File.VPNP12 = data
	case "vpn_sswan":
		s.File.VPNSswan = data
	default:
		return unknown(KindFile, name)
	}
	return nil
}

// Unset drops the entry, and fails where the file lacks it.
func (s *Secrets) Unset(kind, name string) error {
	if !slices.Contains(s.Keys(), kind+" "+name) {
		return fmt.Errorf("%s %s is not in the file", kind, name)
	}
	switch kind {
	case KindToken, KindPassword:
		_ = s.Set(kind, name, "")
	case KindEnv:
		delete(s.Env, name)
	case KindGit:
		delete(s.Git, name)
	case KindFile:
		_ = s.SetFile(name, nil)
	}
	return nil
}

// Keys lists the entries as "KIND NAME", sorted, and never a value.
func (s Secrets) Keys() []string {
	var keys []string
	add := func(kind, name string, held bool) {
		if held {
			keys = append(keys, kind+" "+name)
		}
	}
	add(KindToken, "github", s.Token.GitHub != "")
	add(KindToken, "azure_devops", s.Token.AzureDevOps != "")
	add(KindPassword, "vpn_p12", s.Password.VPNP12 != "")
	add(KindFile, "vpn_p12", len(s.File.VPNP12) > 0)
	add(KindFile, "vpn_sswan", len(s.File.VPNSswan) > 0)
	for name := range s.Env {
		add(KindEnv, name, true)
	}
	for url := range s.Git {
		add(KindGit, url, true)
	}
	slices.Sort(keys)
	return keys
}

func unknown(kind, name string) error {
	switch kind {
	case KindToken, KindEnv, KindGit, KindFile, KindPassword:
		return fmt.Errorf("%s %s: ws has no such entry; see ws mac secrets for the names", kind, name)
	}
	return fmt.Errorf("%s is no kind of entry: want token, env, git, file or password", kind)
}
