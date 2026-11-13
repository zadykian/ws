// Package nvidia is the nvidia role as Go steps. On a machine with NVIDIA's GPU, they install
// NVIDIA's headless driver from Ubuntu's archive, with the modules that Canonical builds and signs
// for the kernel. ws-fans reads the GPU's temperature by its nvidia-smi. See docs/nvidia.md.
package nvidia

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/apt"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// Step names. Each waits on the one before it.
const (
	gpuStep      = "NVIDIA's GPU"
	packagesStep = "NVIDIA's packages"
	driverStep   = "NVIDIA's driver"
)

// Steps returns the role's steps in order: the check of the GPU, the packages, and the note of a
// reboot that loads them. The role takes no values and no secrets. On a machine without NVIDIA's
// GPU the steps after the first plan nothing.
func Steps(_ values.Values, _ secrets.Secrets, m effect.Machine, _ effect.Units, out effect.Out) []step.Step {
	return []step.Step{
		resource.Group{Title: gpuStep, Parts: []resource.Resource{gpu{m, out}}},
		resource.Lazy{Title: packagesStep, After: gpuStep, Parts: gated(m, func(ctx context.Context) ([]resource.Resource, error) {
			names, err := packageNames(ctx, m)
			if err != nil {
				return nil, err
			}
			return []resource.Resource{apt.Packages{M: m, Out: out, Names: names}}, nil
		})},
		resource.Lazy{Title: driverStep, After: packagesStep, Parts: gated(m, func(context.Context) ([]resource.Resource, error) {
			return []resource.Resource{reboot{m, out}}, nil
		})},
	}
}

// gated makes the parts only on a machine with NVIDIA's GPU.
func gated(m effect.Machine, parts func(context.Context) ([]resource.Resource, error)) func(context.Context) ([]resource.Resource, error) {
	return func(ctx context.Context) ([]resource.Resource, error) {
		ok, err := hasGPU(m)
		if err != nil || !ok {
			return nil, err
		}
		return parts(ctx)
	}
}
