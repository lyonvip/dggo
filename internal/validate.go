package internal

import (
	"context"
	"dggo/common"
	"errors"
	"fmt"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/xerror"
	"github.com/melbahja/goph"
	"strconv"
	"strings"
	"sync"
	"time"
)

var KubeMap = sync.Map{}

type Validator struct {
	KubeVIP     string
	SshPasswd   string
	KubeVersion string
	NodePrefix  string
	MasterList  []string
	WorkerList  []string
}

func NewValidator(vip string, sshPasswd string, kubeVersion string, nodePrefix string, masterList []string, workerList []string) *Validator {
	return &Validator{
		KubeVIP:     vip,
		SshPasswd:   sshPasswd,
		KubeVersion: kubeVersion,
		NodePrefix:  nodePrefix,
		MasterList:  masterList,
		WorkerList:  workerList,
	}
}

func (v *Validator) GenVarsPool() {
	kubeVarsPool.HeaderIP = v.MasterList[0]
	kubeVarsPool.HeaderHostname = v.NodePrefix + MasterNodeTag + "1"
	kubeVarsPool.KubeVIP = v.KubeVIP
	kubeVarsPool.KubeVersion = v.KubeVersion
	kubeVarsPool.NerdctlVersion = NerdctlVersion

	// 获取k8s主版本
	parts := strings.Split(v.KubeVersion, ".")
	kubeVarsPool.KubeMainVersion = parts[0] + "." + parts[1]

	// 获取地址与主机名mapping
	for index, ip := range v.MasterList {
		kubeVarsPool.IpHostnameMap[ip] = v.NodePrefix + MasterNodeTag + strconv.Itoa(index)
	}
	if len(v.WorkerList) != 0 {
		for index, ip := range v.WorkerList {
			kubeVarsPool.IpHostnameMap[ip] = v.NodePrefix + WorkerNodeTag + strconv.Itoa(index)
		}
	}
}

func (v *Validator) ValidateIP() error {
	// vip检查
	if err := common.PingTest(v.KubeVIP); err == nil {
		return fmt.Errorf("vip%s已被占用", v.KubeVIP)
	}

	// master节点检查
	/* 地址为空检查 */
	if len(v.MasterList) == 0 {
		return errors.New("master节点不能为0")
	}
	/* 地址重复检查 */
	if len(v.MasterList) > 1 {
		if len(v.MasterList) != len(slice.Unique(v.MasterList)) {
			return errors.New("master节点存在重复ip")
		}
	}
	for _, ip := range v.MasterList {
		/* ping测试 */
		if err := common.PingTest(ip); err != nil {
			return xerror.Wrap(err, fmt.Sprintf("%s地址不通", ip))
		}

		/* ssh连接测试 */
		client, err := common.NewSSHClient(ip, 22, "root", v.SshPasswd)
		if err != nil {
			return err
		}
		KubeMap.Store(ip, client)
	}

	// worker节点检查
	if len(v.WorkerList) != 0 {
		/* 地址重复检查 */
		if len(v.WorkerList) > 1 {
			if len(v.WorkerList) != len(slice.Unique(v.WorkerList)) {
				return errors.New("worker节点存在重复ip")
			}
		}
		for _, ip := range v.WorkerList {
			/* 地址未被master使用 */
			for _, mip := range v.MasterList {
				if mip == ip {
					return fmt.Errorf("%s已被master节点使用", ip)
				}
			}

			/* ping测试 */
			if err := common.PingTest(ip); err != nil {
				return xerror.Wrap(err, fmt.Sprintf("%s地址不通", ip))
			}

			/* ssh连接测试 */
			client, err := common.NewSSHClient(ip, 22, "root", v.SshPasswd)
			if err != nil {
				return xerror.Wrap(err, fmt.Sprintf("%s无法创建ssh连接，请检查密码或网络连接", ip))
			}

			KubeMap.Store(ip, client)
		}
	}

	// 节点可访问外网检查
	timeCtx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()
	var sshPingErr error
	KubeMap.Range(func(key, value any) bool {
		ip := key.(string)
		sshClient := value.(*goph.Client)
		if _, err := sshClient.RunContext(timeCtx, SshPingCmd); err != nil {
			sshPingErr = xerror.Wrap(err, fmt.Sprintf("%s无法访问外网", ip))
			return false
		}
		return true
	})
	if sshPingErr != nil {
		return sshPingErr
	}

	return nil
}

func (v *Validator) ValidateOS() error {
	timeCtx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	for _, ip := range slice.Concat(v.MasterList, v.WorkerList) {
		client, _ := KubeMap.Load(ip)
		sshClient := client.(*goph.Client)

		// 系统名称校验
		osName, err := sshClient.RunContext(timeCtx, SshGetOSNameCmd)
		if err != nil {
			return fmt.Errorf("[%s] 获取系统名称异常", ip)
		}
		osVersionIdList, ok := TestedOSVersion[string(osName)]
		if !ok {
			return fmt.Errorf("[%s] 不支持的系统名称: %s", ip, string(osName))
		}

		// 系统版本号校验
		osVersionId, err := sshClient.RunContext(timeCtx, SshGetOSVersionIDCmd)
		if err != nil {
			return fmt.Errorf("[%s] 获取系统版本号异常", ip)
		}
		if !slice.Contain(osVersionIdList, string(osVersionId)) {
			return fmt.Errorf("[%s] 不支持的系统版本号: %s", ip, string(osVersionId))
		}

		// ubuntu系统自动更新关闭校验
		// 影响： apt安装时可能会pending在系统内核升级
		if string(osName) == "Ubuntu" {
			resAptUpdate, err := sshClient.RunContext(timeCtx, SshCheckAptUpdate)
			if string(resAptUpdate) != "0" || err != nil {
				return fmt.Errorf("[%s] apt自动更新未关闭", ip)
			}
		}
	}

	return nil
}

func (v *Validator) ValidateKubeVersion() error {
	// 验证版本号是否合法: 1.28.11
	parts := strings.Split(v.KubeVersion, ".")
	fn := func(in string) bool {
		if _, err := strconv.Atoi(in); err != nil {
			return false
		}
		return true
	}
	if len(parts) != 3 {
		return errors.New("k8s版本号输入错误")
	}
	for _, part := range parts {
		if !fn(part) {
			return errors.New("k8s版本号输入错误")
		}
	}

	return nil
}
