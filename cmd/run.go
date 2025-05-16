/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"dggo/internal"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/zeromicro/go-zero/core/mr"
	"k8s.io/klog/v2"
	"time"
)

var (
	vip         string
	sshPasswd   string
	kubeVersion string
	nodePrefix  string
	masterList  []string
	workerList  []string
	validator   *internal.Validator
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:     "run",
	Short:   "install k8s via kubeadm",
	PreRunE: k8sPreInstall,
	RunE:    k8sInstall,
}

func k8sPreInstall(cmd *cobra.Command, args []string) error {
	var err error
	validator = internal.NewValidator(vip, sshPasswd, kubeVersion, nodePrefix, masterList, workerList)

	klog.Info("开始检查主机地址...")
	if err = validator.ValidateIP(); err != nil {
		return err
	}
	klog.Info("主机地址无异常")

	klog.Info("开始检查系统版本...")
	if err = validator.ValidateOS(); err != nil {
		return err
	}
	klog.Info("系统版本无异常")

	klog.Info("开始检查k8s版本...")
	if err = validator.ValidateKubeVersion(); err != nil {
		return err
	}
	klog.Info("k8s版本无异常")

	// 生成kubeVarsPool渲染池
	validator.GenVarsPool()

	return nil
}

func k8sInstall(cmd *cobra.Command, args []string) error {
	globalTimeout := 5 * time.Minute
	cmdCaller, err := internal.NewSshCmdCaller(masterList[0])
	if err != nil {
		return err
	}

	// header节点部署
	klog.Info("开始部署header节点...")
	var hasVip bool
	if len(vip) != 0 {
		hasVip = true
	}
	headerInstaller := internal.NewNodeInstaller(
		internal.NewInitNodeStep(true, hasVip, masterList[0], globalTimeout),
		internal.NewInitKubeadmStep(masterList[0], globalTimeout),
		internal.NewInstallAddonsStep(masterList[0], globalTimeout),
	)
	if err = headerInstaller.Run(); err != nil {
		return err
	}
	klog.Info("header节点部署完成...")
	if len(masterList) == 1 && len(workerList) == 0 {
		if err = cmdCaller.UnTaintNode(); err != nil {
			return err
		}
		klog.Info("单节点k8s集群部署完成...")
		return nil
	}

	// 生成join集群命令
	klog.Info("开始生成加入k8s集群命令...")
	var masterJoinCmd, workerJoinCmd string

	if len(masterList) > 1 {
		masterJoinCmd, err = cmdCaller.GenJoinCmd("master")
		if err != nil {
			return err
		}
		klog.Info("生成master节点加入集群命令成功")
		fmt.Println("----------join master command----------")
		fmt.Println()
		fmt.Println(masterJoinCmd)
		fmt.Println()
		fmt.Println("---------------------------------------")
	}

	if len(workerList) > 0 {
		workerJoinCmd, err = cmdCaller.GenJoinCmd("worker")
		if err != nil {
			return err
		}
		klog.Info("生成worker节点加入集群命令成功")
		fmt.Println("----------join worker command----------")
		fmt.Println()
		fmt.Println(workerJoinCmd)
		fmt.Println()
		fmt.Println("---------------------------------------")
	}

	// 非header节点部署
	klog.Info("开始执行其他节点加入集群操作...")
	var runFuncList = make([]func() error, 0)

	if len(masterList) > 1 {
		for _, ip := range masterList[1:] {
			nodeInstaller := internal.NewNodeInstaller(
				internal.NewInitNodeStep(false, hasVip, ip, globalTimeout),
				internal.NewJoinHeaderStep(ip, globalTimeout, masterJoinCmd+internal.SshSetKubectlCmd),
			)
			runFuncList = append(runFuncList, nodeInstaller.Run)
		}
	}

	if len(workerList) > 0 {
		for _, ip := range workerList {
			nodeInstaller := internal.NewNodeInstaller(
				internal.NewInitNodeStep(false, hasVip, ip, globalTimeout),
				internal.NewJoinHeaderStep(ip, globalTimeout, workerJoinCmd+internal.SshSetKubectlCmd),
			)
			runFuncList = append(runFuncList, nodeInstaller.Run)
		}
	}

	if err = mr.Finish(runFuncList...); err != nil {
		return err
	}

	if err = cmdCaller.UnTaintNode(); err != nil {
		return err
	}

	klog.Info("k8s集群部署完成...")
	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringSliceVar(&masterList, "masters", []string{}, "masters ip split by ,")
	runCmd.Flags().StringSliceVar(&workerList, "workers", []string{}, "workers ip split by ,")
	runCmd.Flags().StringVar(&vip, "vip", "", "k8s vip")
	runCmd.Flags().StringVar(&sshPasswd, "ssh-passwd", "", "root ssh password")
	runCmd.Flags().StringVar(&kubeVersion, "kube-version", "1.28.11", "k8s version")
	runCmd.Flags().StringVar(&nodePrefix, "prefix", "k8s-", "node hostname prefix")
	runCmd.Flags().MarkHidden("prefix")
	runCmd.MarkFlagRequired("masters")
	runCmd.MarkFlagRequired("vip")
	runCmd.MarkFlagRequired("ssh-passwd")
}
