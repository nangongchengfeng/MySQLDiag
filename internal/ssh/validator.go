package ssh

import (
	"strings"
)

// Validator 命令验证器
//
// Validator 负责验证SSH命令是否安全，确保只执行白名单中的命令，
// 并且不包含任何危险的子串。
type Validator struct {
	allowedCommands   map[string]struct{}
	blockedSubstrings []string
}

// NewValidator 创建一个新的命令验证器
func NewValidator() *Validator {
	return &Validator{
		allowedCommands:   allowedCommands,
		blockedSubstrings: blockedSubstrings,
	}
}

// Validate 验证命令是否安全
//
// 返回值:
//   - valid: 命令是否有效
//   - reason: 如果无效,返回原因
func (v *Validator) Validate(command string) (valid bool, reason string) {
	cmdLower := strings.ToLower(command)

	// 检查禁止的子串
	for _, blocked := range v.blockedSubstrings {
		if strings.Contains(cmdLower, strings.ToLower(blocked)) {
			return false, "命令包含禁止的子串: " + blocked
		}
	}

	// 解析命令名称
	cmdParts := strings.Fields(command)
	if len(cmdParts) == 0 {
		return false, "空命令"
	}

	cmdName := strings.TrimSuffix(cmdParts[0], ";")

	// 处理带路径的命令
	if strings.Contains(cmdName, "/") {
		parts := strings.Split(cmdName, "/")
		cmdName = parts[len(parts)-1]
	}

	// 特殊处理mysqladmin
	if cmdName == "mysqladmin" {
		allowedMySQLAdmin := map[string]struct{}{
			"ping": {}, "status": {}, "version": {}, "extended-status": {},
			"processlist": {}, "variables": {}, "info": {},
		}
		for subcmd := range allowedMySQLAdmin {
			if strings.Contains(command, subcmd) {
				return true, ""
			}
		}
		return false, "mysqladmin 子命令不在允许列表中"
	}

	// 特殊处理mysqldump
	if cmdName == "mysqldump" {
		if !strings.Contains(command, "--no-data") && !strings.Contains(command, "-d") {
			return false, "mysqldump 必须使用 --no-data 或 -d 参数"
		}
		return true, ""
	}

	// 检查命令是否在白名单中
	if _, ok := v.allowedCommands[cmdName]; !ok {
		return false, "命令不在允许列表中: " + cmdName
	}

	return true, ""
}
