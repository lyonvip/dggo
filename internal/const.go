package internal

const (
	SuperAdminConf       = "/etc/kubernetes/super-admin.conf"
	GenKubeDir           = "mkdir -p /etc/kubernetes"
	RemoteParseDir       = "/usr/local/src"
	NerdctlVersion       = "2.0.5"
	SshCatSuperAdmin     = "cat /etc/kubernetes/admin.conf"
	SshPingCmd           = "ping -c1 -W2 qq.com"
	SshGetOSNameCmd      = "cat /etc/os-release  | grep '^NAME=' | sed 's/.*\"\\(.*\\)\"/\\1/' | tr -d '\\n'"
	SshGetOSVersionIDCmd = "cat /etc/os-release  | grep '^VERSION_ID=' | sed 's/.*\"\\(.*\\)\"/\\1/' | tr -d '\\n'"
	SshCheckAptUpdate    = "grep 1 /etc/apt/apt.conf.d/20auto-upgrades /etc/apt/apt.conf.d/10periodic | wc -l | tr -d '\\n'"
	SshSetKubectlCmd     = "; mkdir -p $HOME/.kube; cp -i /etc/kubernetes/admin.conf $HOME/.kube/config"
	MasterNodeTag        = "master"
	WorkerNodeTag        = "worker"
)

var TestedOSVersion = map[string][]string{
	"Ubuntu": {"22.04"},
	//"openEuler": {"22.03"},
}

var FilenameMap = map[string]string{
	"kubeadm":        "kubeadm.yaml",
	"script":         "k8s-install.sh",
	"calico":         "calico.yaml",
	"metrics-server": "components.yaml",
}
