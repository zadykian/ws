package kind

import (
	"slices"
	"testing"
)

func TestKeepRunningReadsRestartPolicies(t *testing.T) {
	compose := []byte(`services:
  web:
    image: busybox
    restart: unless-stopped
  db:
    image: busybox
    restart: always
  migrate:
    image: busybox
    restart: on-failure
  once:
    image: busybox
    restart: "no"
  bare:
    image: busybox
`)
	got, err := keepRunning(t.Context(), "p", compose)
	if want := []string{"db", "web"}; err != nil || !slices.Equal(got, want) {
		t.Errorf("services = %q, %v; want %q", got, err, want)
	}
}

func TestKeepRunningResolvesAnchors(t *testing.T) {
	compose := []byte(`x-common: &common
  image: busybox
  restart: unless-stopped
services:
  b:
    <<: *common
  a:
    <<: *common
  c:
    image: busybox
    restart: on-failure
`)
	got, err := keepRunning(t.Context(), "p", compose)
	if want := []string{"a", "b"}; err != nil || !slices.Equal(got, want) {
		t.Errorf("services = %q, %v; want %q", got, err, want)
	}
}

func TestKeepRunningNeedsNoEnvFile(t *testing.T) {
	compose := []byte(`services:
  s:
    image: busybox
    env_file:
    - root.env
    environment:
    - KEY=${UNSET_VALUE}
    restart: unless-stopped
`)
	got, err := keepRunning(t.Context(), "p", compose)
	if want := []string{"s"}; err != nil || !slices.Equal(got, want) {
		t.Errorf("services = %q, %v; want %q, with root.env missing", got, err, want)
	}
}

func TestKeepRunningRefusesABadFile(t *testing.T) {
	if _, err := keepRunning(t.Context(), "p", []byte("services: [\n")); err == nil {
		t.Error("a file that is no YAML loaded")
	}
}
