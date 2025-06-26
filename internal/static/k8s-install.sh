#!/bin/bash

echo_log(){
  IP=$1
  Step=$2
  Msg=$3
  echo "[node_ip=${IP}][${Step}] ${Msg}"
}

init_apt_source(){
  Node=$1
  Step="init_apt_source"

  # 配置系统apt源
  echo_log "${Node}" "${Step}" "Config OS sources.list via https://mirrors.aliyun.com/ubuntu/"
  rm -rf /etc/apt/sources.list /etc/apt/sources.list.d/kubernetes.list /etc/apt/keyrings/kubernetes-apt-keyring.gpg
  cat > /etc/apt/sources.list << EOF
deb https://mirrors.aliyun.com/ubuntu/ jammy main restricted universe multiverse
deb-src https://mirrors.aliyun.com/ubuntu/ jammy main restricted universe multiverse

deb https://mirrors.aliyun.com/ubuntu/ jammy-security main restricted universe multiverse
deb-src https://mirrors.aliyun.com/ubuntu/ jammy-security main restricted universe multiverse

deb https://mirrors.aliyun.com/ubuntu/ jammy-updates main restricted universe multiverse
deb-src https://mirrors.aliyun.com/ubuntu/ jammy-updates main restricted universe multiverse

# deb https://mirrors.aliyun.com/ubuntu/ jammy-proposed main restricted universe multiverse
# deb-src https://mirrors.aliyun.com/ubuntu/ jammy-proposed main restricted universe multiverse

deb https://mirrors.aliyun.com/ubuntu/ jammy-backports main restricted universe multiverse
deb-src https://mirrors.aliyun.com/ubuntu/ jammy-backports main restricted universe multiverse
EOF

  # 配置k8s apt源
  echo_log "${Node}" "${Step}" "Config k8s sources.list via https://mirrors.aliyun.com/kubernetes-new/"
  curl -fsSL https://mirrors.aliyun.com/kubernetes-new/core/stable/v{{ .KubeMainVersion }}/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
  echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://mirrors.aliyun.com/kubernetes-new/core/stable/v{{ .KubeMainVersion }}/deb/ /" |
      tee /etc/apt/sources.list.d/kubernetes.list

  # 安装apt-transport-https
  echo_log "${Node}" "${Step}" "Update apt sources"
  apt update
}

install_containerd(){
  Node=$1
  Step="install_containerd"

  # 开启系统模块
  echo_log "${Node}" "${Step}" "Config OS modules-load"

  rm -rf /etc/modules-load.d/containerd.conf
  cat <<EOF > /etc/modules-load.d/containerd.conf
overlay
br_netfilter
EOF
  modprobe overlay
  modprobe br_netfilter

  # 配置内核参数
  echo_log "${Node}" "${Step}" "Config kernel runtime parameters"

  rm -rf /etc/sysctl.d/99-kubernetes-cri.conf
  cat <<EOF > /etc/sysctl.d/99-kubernetes-cri.conf
net.bridge.bridge-nf-call-iptables  = 1
net.ipv4.ip_forward                 = 1
net.bridge.bridge-nf-call-ip6tables = 1
EOF
  sysctl --system

  # 下载二进制文件
  echo_log "${Node}" "${Step}" "Download containerd binary via nerdctl-full package"

  wget -O /usr/local/src/nerdctl-full-{{ .NerdctlVersion }}-linux-amd64.tar.gz \
    https://files.m.daocloud.io/github.com/containerd/nerdctl/releases/download/v{{ .NerdctlVersion }}/nerdctl-full-{{ .NerdctlVersion }}-linux-amd64.tar.gz &>/dev/null
  tar Cxzvvf /usr/local /usr/local/src/nerdctl-full-{{ .NerdctlVersion }}-linux-amd64.tar.gz


  # 配置config.toml
  echo_log "${Node}" "${Step}" "Config containerd parameters via config.toml"

  mkdir -p /etc/containerd
  rm -rf /etc/containerd/config.toml
  cat <<EOF > /etc/containerd/config.toml
disabled_plugins = []
imports = []
oom_score = 0
plugin_dir = ""
required_plugins = []
root = "/var/lib/containerd"
state = "/run/containerd"
temp = ""
version = 2

[cgroup]
  path = ""

[debug]
  address = ""
  format = ""
  gid = 0
  level = ""
  uid = 0

[grpc]
  address = "/run/containerd/containerd.sock"
  gid = 0
  max_recv_message_size = 16777216
  max_send_message_size = 16777216
  tcp_address = ""
  tcp_tls_ca = ""
  tcp_tls_cert = ""
  tcp_tls_key = ""
  uid = 0

[metrics]
  address = ""
  grpc_histogram = false

[plugins]

  [plugins."io.containerd.gc.v1.scheduler"]
    deletion_threshold = 0
    mutation_threshold = 100
    pause_threshold = 0.02
    schedule_delay = "0s"
    startup_delay = "100ms"

  [plugins."io.containerd.grpc.v1.cri"]
    cdi_spec_dirs = ["/etc/cdi", "/var/run/cdi"]
    device_ownership_from_security_context = false
    disable_apparmor = false
    disable_cgroup = false
    disable_hugetlb_controller = true
    disable_proc_mount = false
    disable_tcp_service = true
    drain_exec_sync_io_timeout = "0s"
    enable_cdi = false
    enable_selinux = false
    enable_tls_streaming = false
    enable_unprivileged_icmp = false
    enable_unprivileged_ports = false
    ignore_deprecation_warnings = []
    ignore_image_defined_volumes = false
    image_pull_progress_timeout = "5m0s"
    image_pull_with_sync_fs = false
    max_concurrent_downloads = 3
    max_container_log_line_size = 16384
    netns_mounts_under_state_dir = false
    restrict_oom_score_adj = false
    sandbox_image = "registry.aliyuncs.com/google_containers/pause:3.9"
    selinux_category_range = 1024
    stats_collect_period = 10
    stream_idle_timeout = "4h0m0s"
    stream_server_address = "127.0.0.1"
    stream_server_port = "0"
    systemd_cgroup = false
    tolerate_missing_hugetlb_controller = true
    unset_seccomp_profile = ""

    [plugins."io.containerd.grpc.v1.cri".cni]
      bin_dir = "/opt/cni/bin"
      conf_dir = "/etc/cni/net.d"
      conf_template = ""
      ip_pref = ""
      max_conf_num = 1
      setup_serially = false

    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "runc"
      disable_snapshot_annotations = true
      discard_unpacked_layers = false
      ignore_blockio_not_enabled_errors = false
      ignore_rdt_not_enabled_errors = false
      no_pivot = false
      snapshotter = "overlayfs"

      [plugins."io.containerd.grpc.v1.cri".containerd.default_runtime]
        base_runtime_spec = ""
        cni_conf_dir = ""
        cni_max_conf_num = 0
        container_annotations = []
        pod_annotations = []
        privileged_without_host_devices = false
        privileged_without_host_devices_all_devices_allowed = false
        runtime_engine = ""
        runtime_path = ""
        runtime_root = ""
        runtime_type = ""
        sandbox_mode = ""
        snapshotter = ""

        [plugins."io.containerd.grpc.v1.cri".containerd.default_runtime.options]

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          base_runtime_spec = ""
          cni_conf_dir = ""
          cni_max_conf_num = 0
          container_annotations = []
          pod_annotations = []
          privileged_without_host_devices = false
          privileged_without_host_devices_all_devices_allowed = false
          runtime_engine = ""
          runtime_path = ""
          runtime_root = ""
          runtime_type = "io.containerd.runc.v2"
          sandbox_mode = "podsandbox"
          snapshotter = ""

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            BinaryName = ""
            CriuImagePath = ""
            CriuPath = ""
            CriuWorkPath = ""
            IoGid = 0
            IoUid = 0
            NoNewKeyring = false
            NoPivotRoot = false
            Root = ""
            ShimCgroup = ""
            SystemdCgroup = true

      [plugins."io.containerd.grpc.v1.cri".containerd.untrusted_workload_runtime]
        base_runtime_spec = ""
        cni_conf_dir = ""
        cni_max_conf_num = 0
        container_annotations = []
        pod_annotations = []
        privileged_without_host_devices = false
        privileged_without_host_devices_all_devices_allowed = false
        runtime_engine = ""
        runtime_path = ""
        runtime_root = ""
        runtime_type = ""
        sandbox_mode = ""
        snapshotter = ""

        [plugins."io.containerd.grpc.v1.cri".containerd.untrusted_workload_runtime.options]

    [plugins."io.containerd.grpc.v1.cri".image_decryption]
      key_model = "node"

    [plugins."io.containerd.grpc.v1.cri".registry]
      config_path = ""

      [plugins."io.containerd.grpc.v1.cri".registry.auths]

      [plugins."io.containerd.grpc.v1.cri".registry.configs]

      [plugins."io.containerd.grpc.v1.cri".registry.headers]

      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]

    [plugins."io.containerd.grpc.v1.cri".x509_key_pair_streaming]
      tls_cert_file = ""
      tls_key_file = ""

  [plugins."io.containerd.internal.v1.opt"]
    path = "/opt/containerd"

  [plugins."io.containerd.internal.v1.restart"]
    interval = "10s"

  [plugins."io.containerd.internal.v1.tracing"]

  [plugins."io.containerd.metadata.v1.bolt"]
    content_sharing_policy = "shared"

  [plugins."io.containerd.monitor.v1.cgroups"]
    no_prometheus = false

  [plugins."io.containerd.nri.v1.nri"]
    disable = true
    disable_connections = false
    plugin_config_path = "/etc/nri/conf.d"
    plugin_path = "/opt/nri/plugins"
    plugin_registration_timeout = "5s"
    plugin_request_timeout = "2s"
    socket_path = "/var/run/nri/nri.sock"

  [plugins."io.containerd.runtime.v1.linux"]
    no_shim = false
    runtime = "runc"
    runtime_root = ""
    shim = "containerd-shim"
    shim_debug = false

  [plugins."io.containerd.runtime.v2.task"]
    platforms = ["linux/amd64"]
    sched_core = false

  [plugins."io.containerd.service.v1.diff-service"]
    default = ["walking"]

  [plugins."io.containerd.service.v1.tasks-service"]
    blockio_config_file = ""
    rdt_config_file = ""

  [plugins."io.containerd.snapshotter.v1.aufs"]
    root_path = ""

  [plugins."io.containerd.snapshotter.v1.blockfile"]
    fs_type = ""
    mount_options = []
    root_path = ""
    scratch_file = ""

  [plugins."io.containerd.snapshotter.v1.btrfs"]
    root_path = ""

  [plugins."io.containerd.snapshotter.v1.devmapper"]
    async_remove = false
    base_image_size = ""
    discard_blocks = false
    fs_options = ""
    fs_type = ""
    pool_name = ""
    root_path = ""

  [plugins."io.containerd.snapshotter.v1.native"]
    root_path = ""

  [plugins."io.containerd.snapshotter.v1.overlayfs"]
    mount_options = []
    root_path = ""
    sync_remove = false
    upperdir_label = false

  [plugins."io.containerd.snapshotter.v1.zfs"]
    root_path = ""

  [plugins."io.containerd.tracing.processor.v1.otlp"]

  [plugins."io.containerd.transfer.v1.local"]
    config_path = ""
    max_concurrent_downloads = 3
    max_concurrent_uploaded_layers = 3

    [[plugins."io.containerd.transfer.v1.local".unpack_config]]
      differ = ""
      platform = "linux/amd64"
      snapshotter = "overlayfs"

[proxy_plugins]

[stream_processors]

  [stream_processors."io.containerd.ocicrypt.decoder.v1.tar"]
    accepts = ["application/vnd.oci.image.layer.v1.tar+encrypted"]
    args = ["--decryption-keys-path", "/etc/containerd/ocicrypt/keys"]
    env = ["OCICRYPT_KEYPROVIDER_CONFIG=/etc/containerd/ocicrypt/ocicrypt_keyprovider.conf"]
    path = "ctd-decoder"
    returns = "application/vnd.oci.image.layer.v1.tar"

  [stream_processors."io.containerd.ocicrypt.decoder.v1.tar.gzip"]
    accepts = ["application/vnd.oci.image.layer.v1.tar+gzip+encrypted"]
    args = ["--decryption-keys-path", "/etc/containerd/ocicrypt/keys"]
    env = ["OCICRYPT_KEYPROVIDER_CONFIG=/etc/containerd/ocicrypt/ocicrypt_keyprovider.conf"]
    path = "ctd-decoder"
    returns = "application/vnd.oci.image.layer.v1.tar+gzip"

[timeouts]
  "io.containerd.timeout.bolt.open" = "0s"
  "io.containerd.timeout.metrics.shimstats" = "2s"
  "io.containerd.timeout.shim.cleanup" = "5s"
  "io.containerd.timeout.shim.load" = "5s"
  "io.containerd.timeout.shim.shutdown" = "3s"
  "io.containerd.timeout.task.state" = "2s"

[ttrpc]
  address = ""
  gid = 0
  uid = 0
EOF

  # 启动containerd
  echo_log "${Node}" "${Step}" "Start containerd daemon"
  systemctl enable --now containerd
  if ! systemctl status containerd &>/dev/null; then
    exit 1
  else
    exit 0
  fi
}

install_kube(){
  Node=$1
  Step="install_kube"

  # 安装kubelet/kubeadm/kubectl
  echo_log "${Node}" "${Step}" "Install kubelet/kubeadm/kubectl via apt"
  KUBERNETES_VERSION=$(apt-cache madison kubeadm | grep '{{ .KubeVersion }}' | awk -F '|' '{print $2}' | tr -d ' ')
  apt install -y kubelet=${KUBERNETES_VERSION} kubeadm=${KUBERNETES_VERSION} kubectl=${KUBERNETES_VERSION}
  apt-mark hold kubelet kubeadm kubectl

  echo_log "${Node}" "${Step}" "Start kubelet"
  systemctl enable --now kubelet
}

install_kubevip(){
  Node=$1
  Step="install_kubevip"

  # 配置kubevip
  echo_log "${Node}" "${Step}" "Config kubevip static manifest"
  mkdir -p /etc/kubernetes/manifests
  rm -rf /etc/kubernetes/manifests/kube-vip.yaml
  NicStr=$(echo "{{ .KubeVIP }}" | sed 's/\([0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.\)[0-9]\{1,3\}/\1/')
  NIC=$(ip addr | grep "$NicStr" | awk '{print $NF}')
  cat <<EOF > /etc/kubernetes/manifests/kube-vip.yaml
apiVersion: v1
kind: Pod
metadata:
  name: kube-vip
  namespace: kube-system
spec:
  containers:
  - args:
    - manager
    env:
    - name: vip_arp
      value: "true"
    - name: port
      value: "6443"
    - name: vip_nodename
      valueFrom:
        fieldRef:
          fieldPath: spec.nodeName
    - name: vip_interface
      value: ${NIC}
    - name: vip_cidr
      value: "32"
    - name: dns_mode
      value: first
    - name: cp_enable
      value: "true"
    - name: cp_namespace
      value: kube-system
    - name: svc_enable
      value: "true"
    - name: svc_leasename
      value: plndr-svcs-lock
    - name: vip_leaderelection
      value: "true"
    - name: vip_leasename
      value: plndr-cp-lock
    - name: vip_leaseduration
      value: "5"
    - name: vip_renewdeadline
      value: "3"
    - name: vip_retryperiod
      value: "1"
    - name: lb_enable
      value: "true"
    - name: lb_port
      value: "6443"
    - name: lb_fwdmethod
      value: local
    - name: address
      value: {{ .KubeVIP }}
    - name: prometheus_server
      value: :2112
    image: registry.cn-beijing.aliyuncs.com/lyonkube/kube-vip:v0.8.10
    imagePullPolicy: IfNotPresent
    name: kube-vip
    resources: {}
    securityContext:
      capabilities:
        add:
        - NET_ADMIN
        - NET_RAW
    volumeMounts:
    - mountPath: /etc/kubernetes/admin.conf
      name: kubeconfig
  hostAliases:
  - hostnames:
    - kubernetes
    ip: 127.0.0.1
  hostNetwork: true
  volumes:
  - hostPath:
      {{- if lt .KubeReleaseVersion 29 }}
      path: /etc/kubernetes/admin.conf
      {{- else }}
      path: /etc/kubernetes/super-admin.conf
      {{- end }}
    name: kubeconfig
EOF
}

init_kubeadm(){
  Node=$1
  Step="init_kubeadm"

  # 初始化k8s集群
  echo_log "${Node}" "${Step}" "Init k8s via kubeadm"

  kubeadm init --upload-certs --config /usr/local/src/kubeadm.yaml
  if [ "$?" -ne 0 ]; then
    exit 1
  else
    mkdir -p $HOME/.kube
    cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
    chown $(id -u):$(id -g) $HOME/.kube/config
    exit 0
  fi
}

install_addons(){
  Node=$1
  Step="install_addons"

  # 安装calico插件
  echo_log "${Node}" "${Step}" "Install calico"

  kubectl apply -f /usr/local/src/calico.yaml
  [[ "$?" != "0" ]] && exit 1


  # 安装calico插件
  echo_log "${Node}" "${Step}" "Install metrics-server"
  kubectl apply -f /usr/local/src/components.yaml
  [[ "$?" != "0" ]] && exit 1

  # 等待calico就绪
  echo_log "${Node}" "${Step}" "Wait calico ready..."
  for ((i=1; i<=50; i++)); do
    sleep 3
    kubectl -n kube-system get po | grep 'calico'
    num=$(kubectl -n kube-system get po | grep 'calico' | grep 'Running' | grep '1/1' | wc -l)
    [ "$num" -eq 2 ] && exit 0
  done
  exit 1
}

init_node(){
  Node=$1
  Step="init_node"

  # 关闭防火墙
  echo_log "${Node}" "${Step}" "Disable firewall"
  systemctl disable firewalld --now &>/dev/null
  # 配置主机名
  echo_log "${Node}" "${Step}" "Config hostname"
  hostnamectl set-hostname {{ .LocalHostname }}
  # 配置本地域名解析
  echo_log "${Node}" "${Step}" "Config local dns"
  cat >> /etc/hosts << EOF
{{ if .KubeVIP }}
{{ .KubeVIP }} lb.k8s.local
{{ end }}
{{ range $ip, $hostname := .IpHostnameMap }}
{{ $ip }} {{ $hostname }}
{{- end }}
EOF

  # 关闭swap
  echo_log "${Node}" "${Step}" "Disable swap"
  sed -ri '/\sswap\s/s/^#?/#/' /etc/fstab
  mount -a
  swapoff -a

  # 安装依赖包
  echo_log "${Node}" "${Step}" "Install apt-transport-https/ipset/ipvsadm/chrony via apt"
  apt install -y apt-transport-https ipset ipvsadm chrony

  # 配置时间同步
  echo_log "${Node}" "${Step}" "Config local timedate"
  timedatectl set-timezone Asia/Shanghai
  sed -i '/pool [0-2].ubuntu.pool.ntp.org/d' /etc/chrony/chrony.conf
  sed -i '/pool ntp.ubuntu.com/s/ubuntu/aliyun/' /etc/chrony/chrony.conf
  systemctl restart chronyd

  # 加载内核
  echo_log "${Node}" "${Step}" "Load kernel modules"
  rm -rf /etc/modules-load.d/ipvs.conf
  cat <<EOF > /etc/modules-load.d/ipvs.conf
ip_vs
ip_vs_rr
ip_vs_wrr
ip_vs_sh
nf_conntrack
EOF
  modprobe -- ip_vs
  modprobe -- ip_vs_rr
  modprobe -- ip_vs_wrr
  modprobe -- ip_vs_sh
  modprobe -- nf_conntrack
  if ! lsmod | grep -e ip_vs -e nf_conntrack &>/dev/null; then
    exit 1
  else
    exit 0
  fi

}

gen_master_join_cmd(){
  join_cmd=$(kubeadm token create --print-join-command)
  cert_key=$(kubeadm init phase upload-certs --upload-certs 2>/dev/null | tail -n1)
  res_cmd="${join_cmd} --certificate-key ${cert_key} --control-plane"
  echo "${res_cmd}" | tr -d '\n'
}

gen_worker_join_cmd(){
  join_cmd=$(kubeadm token create --print-join-command)
  echo "${join_cmd}" | tr -d '\n'
}

untaint_master_nodes(){
  Node=$1
  Step="init_node"

  # 删除master污点
  echo_log "${Node}" "${Step}" "Untaint master nodes"
  node_array=($(kubectl get no | grep 'control-plane' | awk '{print $1}'))
  if [ "${#node_array[@]}" -lt "1" ]; then
    exit 1
  fi

  for node in "${node_array[@]}"; do
    if ! kubectl taint no ${node} node-role.kubernetes.io/control-plane:NoSchedule- &>/dev/null; then
      exit 1
    fi
  done
}

node_ip=$2
case $1 in
    "apt")
        init_apt_source "${node_ip}"
        ;;
    "init")
        init_node "${node_ip}"
        ;;
    "containerd")
        install_containerd "${node_ip}"
        ;;
    "kube")
        install_kube "${node_ip}"
        ;;
    "kubevip")
        install_kubevip "${node_ip}"
        ;;
    "kubeadm")
        init_kubeadm "${node_ip}"
        ;;
    "addons")
        install_addons "${node_ip}"
        ;;
    "untaint_node")
        untaint_master_nodes "${node_ip}"
        ;;
    "gen_master_cmd")
        gen_master_join_cmd
        ;;
    "gen_worker_cmd")
        gen_worker_join_cmd
        ;;
esac
