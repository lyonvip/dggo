package internal

import (
	"bytes"
	"dggo/internal/static"
	"embed"
	"errors"
	"fmt"
	"github.com/duke-git/lancet/v2/xerror"
	"github.com/melbahja/goph"
	"path/filepath"
	"sync"
	"text/template"
)

var (
	parserOnce sync.Once
	parser     *fileParser
)

func GetFileParser() *fileParser {
	parserOnce.Do(func() {
		parser = &fileParser{
			embedFS: static.StaticSource,
		}
	})
	return parser
}

type fileParser struct {
	embedFS embed.FS
}

func (f *fileParser) ParseFile(ip, filename string) error {
	sFile, err := f.embedFS.ReadFile(filename)
	if err != nil {
		return xerror.Wrap(err, fmt.Sprintf("获取%s对象异常", filename))
	}

	varsPool := GetVarsPool()
	if varsPool == nil {
		return errors.New("全局渲染变量池未初始化")
	}
	tpl := template.Must(template.New("").Parse(string(sFile)))
	var buff bytes.Buffer
	if err = tpl.Execute(&buff, varsPool); err != nil {
		return xerror.Wrap(err, fmt.Sprintf("渲染%s异常", filename))
	}

	client, _ := KubeMap.Load(ip)
	sshClient := client.(*goph.Client)

	sftp, err := sshClient.NewSftp()
	if err != nil {
		return xerror.Wrap(err, "创建sftp连接异常")
	}

	dFile, err := sftp.Create(filepath.Join(RemoteParseDir, filename))
	if err != nil {
		return xerror.Wrap(err, fmt.Sprintf("[%s] 创建文件对象异常", ip))
	}
	defer dFile.Close()

	if _, err = dFile.Write(buff.Bytes()); err != nil {
		return xerror.Wrap(err, fmt.Sprintf("[%s] 写入渲染内容异常", ip))
	}

	return nil
}
