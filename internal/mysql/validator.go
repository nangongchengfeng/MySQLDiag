package mysql

import (
	"strings"
)

// 允许的SQL语句类型
var allowedStatements = map[string]struct{}{
	"SELECT": {}, "SHOW": {}, "DESCRIBE": {}, "DESC": {}, "EXPLAIN": {}, "USE": {},
	"HELP": {}, "CHECKSUM": {}, "CHECK": {}, "ANALYZE": {},
}

// 禁止的关键字
var blockedKeywords = []string{
	"INSERT", "UPDATE", "DELETE", "REPLACE",
	"DROP", "TRUNCATE", "ALTER", "CREATE", "RENAME",
	"GRANT", "REVOKE",
	"SET",
	"LOAD",
	"LOCK", "UNLOCK",
	"START", "COMMIT", "ROLLBACK", "SAVEPOINT", "RELEASE",
	"PREPARE", "EXECUTE", "DEALLOCATE",
	"OPTIMIZE", "REPAIR", "USE FRM",
	"BACKUP", "RESTORE", "IMPORT", "EXPORT",
	"FLUSH", "RESET", "SHUTDOWN", "KILL", "PURGE", "CHANGE",
	"INSTALL", "UNINSTALL", "PLUGIN",
}

// Validator SQL验证器
//
// Validator 负责验证SQL查询是否安全，确保只执行只读操作。
type Validator struct {
	allowedStatements map[string]struct{}
	blockedKeywords   []string
}

// NewValidator 创建一个新的SQL验证器
func NewValidator() *Validator {
	return &Validator{
		allowedStatements: allowedStatements,
		blockedKeywords:   blockedKeywords,
	}
}

// Validate 验证单条查询是否为只读且安全
//
// 返回值:
//   - valid: 查询是否有效
//   - reason: 如果无效,返回原因
func (v *Validator) Validate(query string) (valid bool, reason string) {
	query = strings.TrimSpace(query)
	if query == "" {
		return false, "空查询"
	}

	queryUpper := strings.ToUpper(query)

	// 移除注释,清理查询
	lines := strings.Split(queryUpper, "\n")
	var cleanLines []string
	for _, line := range lines {
		// 移除 -- 注释
		if idx := strings.Index(line, "--"); idx != -1 {
			line = line[:idx]
		}
		// 移除 # 注释
		if idx := strings.Index(line, "#"); idx != -1 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}
	cleanQuery := strings.Join(cleanLines, " ")

	// 检查禁止的关键字
	for _, keyword := range v.blockedKeywords {
		// 检查关键字是否作为独立单词存在
		if strings.Contains(" "+cleanQuery+" ", " "+keyword+" ") {
			return false, "查询包含禁止的关键字: " + keyword
		}
		// 检查关键字是否在开头
		if strings.HasPrefix(cleanQuery, keyword+" ") {
			return false, "查询以禁止关键字开头: " + keyword
		}
	}

	// 获取第一个词
	parts := strings.Fields(cleanQuery)
	if len(parts) == 0 {
		return false, "空查询"
	}
	firstWord := parts[0]

	// 特殊处理: ANALYZE TABLE 是允许的
	if firstWord == "ANALYZE" && strings.Contains(cleanQuery, "TABLE") {
		return true, ""
	}

	// 检查是否在允许的语句列表中
	if _, ok := v.allowedStatements[firstWord]; !ok {
		return false, "语句类型不在允许列表中: " + firstWord
	}

	return true, ""
}
