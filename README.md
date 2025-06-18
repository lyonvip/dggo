# dggo - 一键安装k8s集群工具

### 准备

- 关闭apt自动更新

  ```bash
  $ sed -i 's/1/0/g' /etc/apt/apt.conf.d/10periodic
  $ sed -i 's/1/0/g' /etc/apt/apt.conf.d/20auto-upgrades
  $ reboot
  ```

- 关闭防火墙

  ```bash
  $ systemctl disable firewalld --now
  ```

- 操作系统目前仅支持ubuntu22.04

### 编译

```bash
$ cd dggo
$ go mod tidy
$ GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o build/dggo .
```

### 安装

#### 参数说明：

```bash
$ dggo run -h
install k8s via kubeadm

Usage:
  dggo run [flags]

Flags:
  -h, --help                  help for run
      --kube-version string   k8s version (default "1.28.11")
      --masters strings       masters ip split by ,
      --ssh-passwd string     root ssh password
      --vip string            k8s vip
      --workers strings       workers ip split by ,
```

- kube-version: 待安装的k8s版本，默认1.28.11，支持1.24-1.33主流发行版本
- masters: master节点地址，多个以逗号分隔
- workers: worker节点地址，多个以逗号分隔
- ssh-passwd: ssh连接密码，默认用户root，所有部署节点需保持密码一致
- vip： 高可用集群虚拟ip

#### 部署示例：

- 单节点集群

  ```bash
  $ dggo run --masters 10.30.2.161 --kube-version 1.29.15 --ssh-passwd Aa123456!
  ```

- 单master多worker

  ```bash
  $ dggo run --masters 10.30.2.161 --workers 10.30.2.162,10.30.2.163 --kube-version 1.29.15 --ssh-passwd Aa123456!
  ```

- 3节点高可用集群

  ```bash
  $ dggo run --masters 10.30.2.161,10.30.2.162,10.30.2.163 --vip 10.30.2.160 --kube-version 1.29.15 --ssh-passwd Aa123456!
  ```

- 3节点多worker高可用集群

  ```bash
  $ dggo run --masters 10.30.2.161,10.30.2.162,10.30.2.163 --workers 10.30.2.164,10.30.2.165 --vip 10.30.2.160 --kube-version 1.29.15 --ssh-passwd Aa123456!
  ```

  