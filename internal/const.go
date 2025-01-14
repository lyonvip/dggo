package internal

const (
	RemoteParseDir       = "/usr/local/src"
	NerdctlVersion       = "1.7.7"
	SshPingCmd           = "ping -c1 -W2 mirrors.tuna.tsinghua.edu.cn"
	SshGetOSNameCmd      = "cat /etc/os-release  | grep '^NAME=' | sed 's/.*\"\\(.*\\)\"/\\1/'"
	SshGetOSVersionIDCmd = "cat /etc/os-release  | grep '^VERSION_ID=' | sed 's/.*\"\\(.*\\)\"/\\1/'"
	SshCheckAptUpdate    = "grep 1 /etc/apt/apt.conf.d/20auto-upgrades /etc/apt/apt.conf.d/20auto-upgrades | wc -l"
	SshSetKubectlCmd = "; mkdir -p $HOME/.kube; cp -i /etc/kubernetes/admin.conf $HOME/.kube/config"
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
