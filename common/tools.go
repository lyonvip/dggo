package common

import (
	"github.com/duke-git/lancet/v2/xerror"
	"github.com/go-ping/ping"
	"github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
	"time"
)

// NewSSHClient ssh客户端
func NewSSHClient(host string, port uint, username, password string) (*goph.Client, error) {
	goph.DefaultKnownHosts()
	client, err := goph.NewConn(&goph.Config{
		User:     username,
		Addr:     host,
		Port:     port,
		Auth:     goph.Password(password),
		Timeout:  goph.DefaultTimeout,
		Callback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return nil, xerror.Wrap(err, host+"无法创建ssh连接，请检查密码或网络连接")
	}
	return client, nil
}

// PingTest ping包测试
func PingTest(host string) error {
	pinger, err := ping.NewPinger(host)
	if err != nil {
		return err
	}
	pinger.SetPrivileged(true)
	pinger.Count = 2
	pinger.Timeout = 3 * time.Second
	if err = pinger.Run(); err != nil {
		return err
	}
	return nil
}
