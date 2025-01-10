package internal

import (
	"context"
	"fmt"
	"github.com/duke-git/lancet/v2/xerror"
	"github.com/melbahja/goph"
	"k8s.io/klog/v2"
	"path/filepath"
	"time"
)

type NodeStep interface {
	Name() string
	Addr() string
	Setup() error
}

// InitNodeStep 节点初始化
type InitNodeStep struct {
	name     string
	isHeader bool
	ip       string
	timeout  time.Duration
}

func NewInitNodeStep(isHeader bool, ip string, timeout time.Duration) NodeStep {
	stepName := "节点初始化"
	return &InitNodeStep{stepName, isHeader, ip, timeout}
}

func (i *InitNodeStep) Setup() error {
	client, _ := KubeMap.Load(i.ip)
	sshClient := client.(*goph.Client)
	timeCtx, cancel := context.WithTimeout(context.TODO(), i.timeout)
	defer cancel()

	myFilenameParser := GetFileParser()

	// 渲染静态文件：如果不是header，仅需要渲染scripts，否则需要渲染所有static静态文件
	if err := myFilenameParser.ParseFile(i.ip, FilenameMap["script"]); err != nil {
		return err
	}
	if i.isHeader {
		for tag, filename := range FilenameMap {
			if tag != "script" {
				if err := myFilenameParser.ParseFile(i.ip, filename); err != nil {
					return err
				}
			}
		}
	}

	// 执行初始化
	shellScript := filepath.Join(RemoteParseDir, FilenameMap["script"])
	initCmdList := []string{
		fmt.Sprintf("bash %s apt", shellScript),
		fmt.Sprintf("bash %s init", shellScript),
		fmt.Sprintf("bash %s containerd", shellScript),
		fmt.Sprintf("bash %s kube", shellScript),
		fmt.Sprintf("bash %s kubevip", shellScript),
	}
	for _, cmd := range initCmdList {
		if _, err := sshClient.RunContext(timeCtx, cmd); err != nil {
			return err
		}
	}

	return nil
}

func (i *InitNodeStep) Name() string {
	return i.name
}

func (i *InitNodeStep) Addr() string {
	return i.ip
}

// InitKubeadmStep kubeadm初始化
type InitKubeadmStep struct {
	name    string
	ip      string
	timeout time.Duration
}

func NewInitKubeadmStep(ip string, timeout time.Duration) NodeStep {
	stepName := "kubeadm初始化"
	return &InitKubeadmStep{stepName, ip, timeout}
}

func (i *InitKubeadmStep) Setup() error {
	client, _ := KubeMap.Load(i.ip)
	sshClient := client.(*goph.Client)
	timeCtx, cancel := context.WithTimeout(context.TODO(), i.timeout)
	defer cancel()

	shellScript := filepath.Join(RemoteParseDir, FilenameMap["script"])
	initCmd := fmt.Sprintf("bash %s kubeadm", shellScript)
	if _, err := sshClient.RunContext(timeCtx, initCmd); err != nil {
		return err
	}

	return nil
}

func (i *InitKubeadmStep) Name() string {
	return i.name
}

func (i *InitKubeadmStep) Addr() string {
	return i.ip
}

// InstallAddonsStep 插件安装
type InstallAddonsStep struct {
	name    string
	ip      string
	timeout time.Duration
}

func NewInstallAddonsStep(ip string, timeout time.Duration) NodeStep {
	stepName := "插件安装"
	return &InstallAddonsStep{stepName, ip, timeout}
}

func (i *InstallAddonsStep) Setup() error {
	client, _ := KubeMap.Load(i.ip)
	sshClient := client.(*goph.Client)
	timeCtx, cancel := context.WithTimeout(context.TODO(), i.timeout)
	defer cancel()

	shellScript := filepath.Join(RemoteParseDir, FilenameMap["script"])
	installCmd := fmt.Sprintf("bash %s addons", shellScript)
	if _, err := sshClient.RunContext(timeCtx, installCmd); err != nil {
		return err
	}

	return nil
}

func (i *InstallAddonsStep) Name() string {
	return i.name
}

func (i *InstallAddonsStep) Addr() string {
	return i.ip
}

type NodeInstaller struct {
	Ticker *time.Ticker
	Signal bool
	ErrCh  chan error
	Steps  []NodeStep
}

func NewNodeInstaller(steps ...NodeStep) *NodeInstaller {
	ticker := time.NewTicker(3 * time.Second)
	return &NodeInstaller{
		Ticker: ticker,
		Signal: false,
		ErrCh:  make(chan error),
		Steps:  steps,
	}
}

func (n *NodeInstaller) Run(ctx context.Context) error {
	defer n.Ticker.Stop()
	for _, item := range n.Steps {
		fooStep := item
		stepName := fooStep.Name()
		addr := fooStep.Addr()
		klog.Infof("[%s] 开始执行%s>>>", stepName, addr)
		go func() {
			n.ErrCh <- fooStep.Setup()
		}()
		n.Signal = false
		for !n.Signal {
			select {
			case <-n.Ticker.C:
				klog.Infof("[%s] 正在执行%s...", stepName, addr)
			case err := <-n.ErrCh:
				if err != nil {
					return xerror.Wrap(err, fmt.Sprintf("[%s] 执行%s出现异常", stepName, addr))
				}
				n.Signal = true
			}
		}
		klog.Infof("[%s] %s执行完毕<<<", stepName, addr)
	}

	return nil
}
