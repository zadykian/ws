package docker

type daemonJSON struct {
	Bip                 string                       `json:"bip"`
	Builder             builder                      `json:"builder"`
	CgroupParent        string                       `json:"cgroup-parent"`
	DefaultAddressPools []addressPool                `json:"default-address-pools"`
	DefaultNetworkOpts  map[string]map[string]string `json:"default-network-opts"`
	DNS                 []string                     `json:"dns"`
	IP                  string                       `json:"ip"`
	LiveRestore         bool                         `json:"live-restore"`
	LogDriver           string                       `json:"log-driver"`
	LogOpts             map[string]string            `json:"log-opts"`
}

type builder struct {
	GC struct {
		Enabled bool       `json:"enabled"`
		Policy  []gcPolicy `json:"policy"`
	} `json:"gc"`
}

type gcPolicy struct {
	All           bool     `json:"all,omitempty"`
	Filter        []string `json:"filter,omitempty"`
	ReservedSpace string   `json:"reservedSpace,omitempty"`
	MaxUsedSpace  string   `json:"maxUsedSpace,omitempty"`
	MinFreeSpace  string   `json:"minFreeSpace,omitempty"`
}

type addressPool struct {
	Base string `json:"base"`
	Size int    `json:"size"`
}
