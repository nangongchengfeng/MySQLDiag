// Package ssh 提供SSH远程命令执行功能
//
// 该包提供安全的SSH连接管理和只读命令执行功能,
// 包含命令白名单验证机制,确保只能执行安全的只读命令。
package ssh

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// Client SSH客户端
//
// Client 负责管理SSH连接和执行远程命令。
// 所有命令都会经过白名单验证,确保只执行安全的只读操作。
type Client struct {
	host     string
	port     int
	username string
	password *string
	keyPath  *string
	client   *ssh.Client
	validator *Validator
	logger   *slog.Logger
}

// NewClient 创建一个新的SSH客户端
//
// 参数:
//   - host: SSH主机地址
//   - port: SSH端口
//   - username: SSH用户名
//   - password: SSH密码(可选)
//   - keyPath: SSH密钥路径(可选)
//
// 返回: 新的SSH客户端实例
func NewClient(host string, port int, username string, password, keyPath *string) *Client {
	return &Client{
		host:      host,
		port:      port,
		username:  username,
		password:  password,
		keyPath:   keyPath,
		validator: NewValidator(),
		logger:    slog.Default(),
	}
}

// Connect 建立SSH连接
//
// 如果已连接则不做任何操作。
// 优先使用密钥认证,如果密钥不可用则使用密码认证。
func (c *Client) Connect() error {
	if c.client != nil {
		return nil
	}

	var authMethods []ssh.AuthMethod

	// 尝试使用密钥认证
	if c.keyPath != nil {
		if signer, err := c.loadPrivateKey(*c.keyPath); err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
			c.logger.Debug("已添加SSH密钥认证")
		} else {
			c.logger.Warn("加载SSH密钥失败,将尝试密码认证", "error", err)
		}
	}

	// 添加密码认证(如果提供)
	if c.password != nil {
		authMethods = append(authMethods, ssh.Password(*c.password))
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("未提供SSH认证方式(密码或密钥)")
	}

	config := &ssh.ClientConfig{
		User:            c.username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	c.logger.Info("正在连接SSH", "addr", addr, "user", c.username)

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %w", err)
	}

	c.client = client
	c.logger.Info("SSH连接成功")
	return nil
}

// Disconnect 关闭SSH连接
func (c *Client) Disconnect() {
	if c.client != nil {
		c.client.Close()
		c.client = nil
		c.logger.Info("SSH连接已断开")
	}
}

// Execute 执行只读命令
//
// 参数:
//   - command: 要执行的命令
//   - timeout: 超时时间(秒)
//
// 返回: 命令输出,或错误信息
func (c *Client) Execute(command string, timeout int) (string, error) {
	// 验证命令
	valid, reason := c.validator.Validate(command)
	if !valid {
		return "", fmt.Errorf("命令验证失败: %s", reason)
	}

	// 确保已连接
	if c.client == nil {
		if err := c.Connect(); err != nil {
			return "", err
		}
	}

	c.logger.Info("执行SSH命令", "command", command)

	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// 设置超时
	if timeout <= 0 {
		timeout = 60
	}
	done := make(chan error, 1)

	go func() {
		done <- session.Run(command)
	}()

	select {
	case err = <-done:
		// 命令完成
	case <-time.After(time.Duration(timeout) * time.Second):
		session.Signal(ssh.SIGKILL)
		return "", fmt.Errorf("命令执行超时(%d秒)", timeout)
	}

	output := stdout.String()
	errOutput := stderr.String()

	if err != nil {
		c.logger.Warn("命令执行异常", "error", err, "stderr", errOutput)
	}

	result := output
	if errOutput != "" && output == "" {
		result = errOutput
	} else if errOutput != "" {
		result = output + "\n\nSTDERR:\n" + errOutput
	}

	return result, nil
}

// loadPrivateKey 加载SSH私钥
func (c *Client) loadPrivateKey(path string) (ssh.Signer, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(keyBytes)
}

// SetLogger 设置日志记录器
func (c *Client) SetLogger(logger *slog.Logger) {
	c.logger = logger
}
