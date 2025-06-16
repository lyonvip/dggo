package internal

import (
	"context"
	"fmt"
	"github.com/melbahja/goph"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type NodeStep interface {
	Setup() error
}

// InitNodeStep 节点初始化
type InitNodeStep struct {
	isHeader bool
	hasVip   bool
	ip       string
}

func NewInitNodeStep(isHeader bool, hasVip bool, ip string) NodeStep {
	return &InitNodeStep{isHeader, hasVip, ip}
}

func (i *InitNodeStep) Setup() error {
	cmdCaller, err := NewSshCmdCaller(i.ip)
	if err != nil {
		return err
	}

	myFilenameParser := GetFileParser()

	// 渲染静态文件：如果不是header，仅需要渲染scripts，否则需要渲染所有static静态文件
	if err = myFilenameParser.ParseFile(i.ip, FilenameMap["script"]); err != nil {
		return err
	}
	if i.isHeader {
		for tag, filename := range FilenameMap {
			if tag != "script" {
				if err = myFilenameParser.ParseFile(i.ip, filename); err != nil {
					return err
				}
			}
		}
	}

	// 执行初始化
	shellScript := filepath.Join(RemoteParseDir, FilenameMap["script"])
	initCmdList := []string{
		fmt.Sprintf("bash %s apt %s", shellScript, i.ip),
		fmt.Sprintf("bash %s init %s", shellScript, i.ip),
		fmt.Sprintf("bash %s containerd %s", shellScript, i.ip),
		fmt.Sprintf("bash %s kube %s", shellScript, i.ip),
	}

	if i.hasVip {
		initCmdList = append(initCmdList, fmt.Sprintf("bash %s kubevip %s", shellScript, i.ip))
	}

	for _, cmd := range initCmdList {
		if err = cmdCaller.RemoteShell(cmd); err != nil {
			return err
		}
	}

	return nil
}

// InitKubeadmStep kubeadm初始化
type InitKubeadmStep struct {
	ip string
}

func NewInitKubeadmStep(ip string) NodeStep {
	return &InitKubeadmStep{ip}
}

func (i *InitKubeadmStep) Setup() error {
	cmdCaller, err := NewSshCmdCaller(i.ip)
	if err != nil {
		return err
	}

	shellScript := filepath.Join(RemoteParseDir, FilenameMap["script"])
	initCmd := fmt.Sprintf("bash %s kubeadm %s", shellScript, i.ip)
	if err = cmdCaller.RemoteShell(initCmd); err != nil {
		return err
	}

	return nil
}

// InstallAddonsStep 插件安装
type InstallAddonsStep struct {
	ip string
}

func NewInstallAddonsStep(ip string) NodeStep {
	return &InstallAddonsStep{ip}
}

func (i *InstallAddonsStep) Setup() error {
	cmdCaller, err := NewSshCmdCaller(i.ip)
	if err != nil {
		return err
	}

	shellScript := filepath.Join(RemoteParseDir, FilenameMap["script"])
	installCmd := fmt.Sprintf("bash %s addons %s", shellScript, i.ip)
	if err = cmdCaller.RemoteShell(installCmd); err != nil {
		return err
	}

	return nil
}

type JoinHeaderStep struct {
	ip      string
	joinCmd string
}

func NewJoinHeaderStep(ip string, joinCmd string) NodeStep {
	return &JoinHeaderStep{ip, joinCmd}
}

func (j *JoinHeaderStep) Setup() error {
	cmdCaller, err := NewSshCmdCaller(j.ip)
	if err != nil {
		return err
	}

	if err = cmdCaller.RemoteShell(j.joinCmd); err != nil {
		return err
	}

	return nil
}

/* 执行安装对象 */
type NodeInstaller struct {
	Signal bool
	ErrCh  chan error
	Steps  []NodeStep
}

func NewNodeInstaller(steps ...NodeStep) *NodeInstaller {
	return &NodeInstaller{
		Signal: false,
		ErrCh:  make(chan error),
		Steps:  steps,
	}
}

func (n *NodeInstaller) Run() error {
	for _, item := range n.Steps {
		fooStep := item
		go func() {
			n.ErrCh <- fooStep.Setup()
		}()
		n.Signal = false
		for !n.Signal {
			select {
			case err := <-n.ErrCh:
				if err != nil {
					return err
				}
				n.Signal = true
			}
		}
	}

	return nil
}

type SshCmdCaller struct {
	sshClient *goph.Client
	wg        *sync.WaitGroup
	ip        string
}

func NewSshCmdCaller(ip string) (*SshCmdCaller, error) {
	client, ok := KubeMap.Load(ip)
	if !ok {
		return nil, fmt.Errorf("无法获取%s的ssh连接", ip)
	}
	sshClient := client.(*goph.Client)
	return &SshCmdCaller{sshClient, new(sync.WaitGroup), ip}, nil
}

func (g *SshCmdCaller) GenJoinCmd(role string) (string, error) {
	var execCmd string
	shellScript := filepath.Join(RemoteParseDir, FilenameMap["script"])
	timeCtx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	switch role {
	case "master":
		execCmd = fmt.Sprintf("bash %s gen_master_cmd", shellScript)
	case "worker":
		execCmd = fmt.Sprintf("bash %s gen_worker_cmd", shellScript)
	}
	res, err := g.sshClient.RunContext(timeCtx, execCmd)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func (g *SshCmdCaller) GenKubeConfig() ([]byte, error) {
	timeCtx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	res, err := g.sshClient.RunContext(timeCtx, SshCatSuperAdmin)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (g *SshCmdCaller) UnTaintNode() error {
	shellScript := filepath.Join(RemoteParseDir, FilenameMap["script"])
	execCmd := fmt.Sprintf("bash %s untaint_node %s", shellScript, g.ip)
	if err := g.RemoteShell(execCmd); err != nil {
		return err
	}
	return nil
}

func (g *SshCmdCaller) RemoteShell(cmd string) error {
	session, err := g.sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	if err = session.Start(cmd); err != nil {
		return err
	}

	g.wg.Add(2)
	go func() {
		defer g.wg.Done()
		io.Copy(os.Stdout, stdout)
	}()
	go func() {
		defer g.wg.Done()
		io.Copy(os.Stderr, stderr)
	}()

	g.wg.Wait()

	if err = session.Wait(); err != nil {
		return err
	}

	return nil
}

func (g *SshCmdCaller) RemoteWriteFile(content []byte) error {
	if err := g.RemoteShell(GenKubeDir); err != nil {
		return err
	}

	sftpClient, err := g.sshClient.NewSftp()
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	fi, err := sftpClient.OpenFile(SuperAdminConf, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer fi.Close()

	if _, err = fi.Write(content); err != nil {
		return err
	}

	return nil
}
