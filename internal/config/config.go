// Package config 提供配置管理功能
//
// 该包负责从环境变量加载配置，并提供配置数据结构。
// 支持 .env 文件加载和环境变量覆盖。
package config

import (
	"fmt"

	"github.com/joho/godotenv"
)

// Config 包含所有配置项
//
// Config 结构体聚合了API、SSH、MySQL和诊断相关的所有配置，
// 提供类型安全的配置访问。
type Config struct {
	// API配置
	APIKey  string // API密钥,用于访问LLM服务
	BaseURL string // API基础URL
	Model   string // 使用的模型名称

	// SSH配置
	SSHHost     string  // SSH主机地址
	SSHPort     int     // SSH端口,默认22
	SSHUser     string  // SSH用户名,默认root
	SSHPassword *string // SSH密码(可选)
	SSHKeyPath  *string // SSH密钥路径(可选)

	// MySQL配置
	MySQLHost     string  // MySQL主机地址
	MySQLPort     int     // MySQL端口,默认3306
	MySQLUser     string  // MySQL用户名,默认root
	MySQLPassword string  // MySQL密码
	MySQLDatabase *string // MySQL数据库名(可选)

	// 诊断配置
	SlowQueryThreshold float64 // 慢查询阈值(秒),默认0.5
}

// Load 从环境变量加载配置
//
// 该函数会:
// 1. 尝试加载 .env 文件(如果存在)
// 2. 从环境变量读取所有配置项
// 3. 验证必填配置项
// 4. 设置默认值
//
// 返回加载好的Config对象,或错误信息
func Load() (*Config, error) {
	// 尝试加载.env文件,失败不报错
	_ = godotenv.Load()

	cfg := &Config{}

	// API配置
	cfg.APIKey = getEnvRequired("SILICONFLOW_API_KEY")
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("缺少必需的环境变量: SILICONFLOW_API_KEY")
	}

	cfg.BaseURL = getEnvWithDefault("SILICONFLOW_BASE_URL", "https://api.siliconflow.cn/v1/")
	cfg.Model = getEnvWithDefault("SILICONFLOW_MODEL", "deepseek-ai/DeepSeek-V3.2")

	// SSH配置
	cfg.SSHHost = getEnvRequired("SSH_HOST")
	if cfg.SSHHost == "" {
		return nil, fmt.Errorf("缺少必需的环境变量: SSH_HOST")
	}

	sshPort, err := getEnvAsInt("SSH_PORT", 22)
	if err != nil {
		return nil, fmt.Errorf("SSH_PORT 格式错误: %w", err)
	}
	cfg.SSHPort = sshPort

	cfg.SSHUser = getEnvWithDefault("SSH_USER", "root")
	cfg.SSHPassword = getEnvPtr("SSH_PASSWORD")
	cfg.SSHKeyPath = getEnvPtr("SSH_KEY_PATH")

	// MySQL配置
	cfg.MySQLHost = getEnvRequired("MYSQL_HOST")
	if cfg.MySQLHost == "" {
		return nil, fmt.Errorf("缺少必需的环境变量: MYSQL_HOST")
	}

	mysqlPort, err := getEnvAsInt("MYSQL_PORT", 3306)
	if err != nil {
		return nil, fmt.Errorf("MYSQL_PORT 格式错误: %w", err)
	}
	cfg.MySQLPort = mysqlPort

	cfg.MySQLUser = getEnvWithDefault("MYSQL_USER", "root")
	cfg.MySQLPassword = getEnvWithDefault("MYSQL_PASSWORD", "")
	cfg.MySQLDatabase = getEnvPtr("MYSQL_DATABASE")

	// 诊断配置
	slowQueryThreshold, err := getEnvAsFloat("SLOW_QUERY_THRESHOLD", 0.5)
	if err != nil {
		return nil, fmt.Errorf("SLOW_QUERY_THRESHOLD 格式错误: %w", err)
	}
	cfg.SlowQueryThreshold = slowQueryThreshold

	return cfg, nil
}

// SSHAddr 返回完整的SSH地址,格式为 host:port
func (c *Config) SSHAddr() string {
	return fmt.Sprintf("%s:%d", c.SSHHost, c.SSHPort)
}

// MySQLAddr 返回完整的MySQL地址,格式为 host:port
func (c *Config) MySQLAddr() string {
	return fmt.Sprintf("%s:%d", c.MySQLHost, c.MySQLPort)
}

// MySQLDSN 返回MySQL连接的DSN字符串
// 格式为: user:password@tcp(host:port)/database?charset=utf8mb4
func (c *Config) MySQLDSN() string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/",
		c.MySQLUser, c.MySQLPassword, c.MySQLHost, c.MySQLPort)
	if c.MySQLDatabase != nil && *c.MySQLDatabase != "" {
		dsn += *c.MySQLDatabase
	}
	dsn += "?charset=utf8mb4&parseTime=True&loc=Local"
	return dsn
}
