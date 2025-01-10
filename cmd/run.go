/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"dggo/internal"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
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
	// header节点部署
	localTimeout := 5 * time.Minute
	headerNode := internal.NewInitNodeStep(true, masterList[0], localTimeout)
	headerKubeadm := internal.NewInitKubeadmStep(masterList[0], localTimeout)
	headerAddons := internal.NewInstallAddonsStep(masterList[0], localTimeout)
	headerInstaller := internal.NewNodeInstaller(headerNode, headerKubeadm, headerAddons)
	if err := headerInstaller.Run(context.TODO()); err != nil {
		return err
	}

	// Todo: 生成join命令

	// 非header节点初始化
	var needInitNodeIP = make([]string, 0)
	if len(masterList) > 1 {
		needInitNodeIP = append(needInitNodeIP, masterList[1:]...)
	}
	if len(workerList) > 0 {
		needInitNodeIP = append(needInitNodeIP, workerList...)
	}

	g, ctx := errgroup.WithContext(context.TODO())

	if len(masterList) > 1 {
		for _, ip := range masterList[1:] {
			g.Go(func() error {
				nodeInstaller := internal.NewNodeInstaller(internal.NewInitNodeStep(false, ip, localTimeout))
				// 节点初始化
				if err := nodeInstaller.Run(ctx); err != nil {
					return err
				}
				return nil
			})
		}
	}

	if len(workerList) > 0 {
		for _, ip := range workerList {
			g.Go(func() error {
				nodeInstaller := internal.NewNodeInstaller(internal.NewInitNodeStep(false, ip, localTimeout))
				// 节点初始化
				if err := nodeInstaller.Run(ctx); err != nil {
					return err
				}
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		return err
	}

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
