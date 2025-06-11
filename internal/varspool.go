package internal

import (
	"errors"
)

var kubeVarsPool = NewVarsPool()

type VarsPool struct {
	HeaderIP           string
	HeaderHostname     string
	KubeVIP            string
	KubeVersion        string
	KubeMainVersion    string
	KubeReleaseVersion int
	NerdctlVersion     string
	LocalIp            string
	LocalHostname      string
	IpHostnameMap      map[string]string
}

func (v *VarsPool) GenLocalHostname(ip string) error {
	if len(v.IpHostnameMap[ip]) == 0 {
		return errors.New("ip not found in vars pool")
	}
	v.LocalHostname = v.IpHostnameMap[ip]
	return nil
}

func NewVarsPool() *VarsPool {
	return &VarsPool{
		IpHostnameMap: make(map[string]string),
	}
}

func GetVarsPool() *VarsPool {
	return kubeVarsPool
}
