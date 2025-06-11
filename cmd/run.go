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

	klog.Info("Validate all ip")
	if err = validator.ValidateIP(); err != nil {
		return err
	}

	klog.Info("Validate OS Version")
	if err = validator.ValidateOS(); err != nil {
		return err
	}

	klog.Info("Validate K8s Version")
	if err = validator.ValidateKubeVersion(); err != nil {
		return err
	}

	// 生成kubeVarsPool渲染池
	klog.Info("Generate kubeVarsPool")
	validator.GenVarsPool()

	return nil
}

func k8sInstall(cmd *cobra.Command, args []string) error {
	headerCaller, err := internal.NewSshCmdCaller(masterList[0])
	if err != nil {
		return err
	}

	// 安装header节点
	klog.Info("Startup header node")
	var hasVip bool
	if len(vip) != 0 {
		hasVip = true
	}
	headerInstaller := internal.NewNodeInstaller(
		internal.NewInitNodeStep(true, hasVip, masterList[0]),
		internal.NewInitKubeadmStep(masterList[0]),
		internal.NewInstallAddonsStep(masterList[0]),
	)
	if err = headerInstaller.Run(); err != nil {
		return err
	}
	klog.Info("Header node startup completed")
	if len(masterList) == 1 && len(workerList) == 0 {
		if err = headerCaller.UnTaintNode(); err != nil {
			return err
		}
		klog.Info("single-node-k8s deployment completed")
		return nil
	}

	// 生成join集群命令
	var masterJoinCmd, workerJoinCmd string

	if len(masterList) > 1 {
		masterJoinCmd, err = headerCaller.GenJoinCmd("master")
		if err != nil {
			return err
		}
		fmt.Println("[master join command] ", masterJoinCmd)
	}

	if len(workerList) > 0 {
		workerJoinCmd, err = headerCaller.GenJoinCmd("worker")
		if err != nil {
			return err
		}
		fmt.Println("[worker join command] ", workerJoinCmd)
	}

	// 非header节点部署
	klog.Info("Nodes start join header")
	var runFuncList = make([]func() error, 0)

	if len(masterList) > 1 {
		for _, ip := range masterList[1:] {
			nodeInstaller := internal.NewNodeInstaller(
				internal.NewInitNodeStep(false, hasVip, ip),
				internal.NewJoinHeaderStep(ip, masterJoinCmd+internal.SshSetKubectlCmd),
			)
			runFuncList = append(runFuncList, nodeInstaller.Run)
		}
	}

	if len(workerList) > 0 {
		for _, ip := range workerList {
			nodeInstaller := internal.NewNodeInstaller(
				internal.NewInitNodeStep(false, hasVip, ip),
				internal.NewJoinHeaderStep(ip, workerJoinCmd+internal.SshSetKubectlCmd),
			)
			runFuncList = append(runFuncList, nodeInstaller.Run)
		}
	}

	if err = mr.Finish(runFuncList...); err != nil {
		return err
	}

	if err = headerCaller.UnTaintNode(); err != nil {
		return err
	}

	klog.Info("k8s cluster deployment completed")
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
	//runCmd.MarkFlagRequired("vip")
	runCmd.MarkFlagRequired("ssh-passwd")
}
