package internal

var kubeVarsPool *VarsPool

type VarsPool struct {
	HeaderIP        string
	HeaderHostname  string
	KubeVIP         string
	KubeVersion     string
	KubeMainVersion string
	NerdctlVersion  string
	IpHostnameMap   map[string]string
}

func GetVarsPool() *VarsPool {
	return kubeVarsPool
}
