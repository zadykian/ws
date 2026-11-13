package nvidia

// branch is the driver's branch in Ubuntu's archive, the one that `ubuntu-drivers install --gpgpu`
// picks for the RTX 5080. NVIDIA's open modules alone drive that GPU.
const branch = "595-server"

// smiPackage holds nvidia-smi, which ws-fans reads the GPU's temperature by.
const smiPackage = "nvidia-utils-" + branch

// driverPackages are the driver's userspace, without X, and nvidia-smi. Canonical builds and
// signs the modules, so no DKMS builds them.
var driverPackages = []string{"nvidia-headless-no-dkms-" + branch + "-open", smiPackage}

// modulesPrefix, with a kernel's flavor, names the metapackage that follows that flavor's kernels
// with their modules.
const modulesPrefix = "linux-modules-nvidia-" + branch + "-open-"

// flavors are the generic kernels' flavors that Ubuntu builds the modules for, on amd64 and arm64.
var flavors = []string{"generic", "generic-7.0", "generic-hwe-26.04", "generic-hwe-26.04-edge"}

// allNames are the driver's packages and every flavor's modules.
func allNames() []string {
	names := append([]string{}, driverPackages...)
	for _, f := range flavors {
		names = append(names, modulesPrefix+f)
	}
	return names
}
