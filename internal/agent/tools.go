package agent

import (
	"context"
	"fmt"
	"strings"
)

// executeTool 执行工具调用
func (a *Agent) executeTool(ctx context.Context, toolCall *ToolCallImpl) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	switch {
	case toolCall.IsMySQLQuery():
		return a.executeMySQLQuery(toolCall.Input)
	case toolCall.IsSSHExec():
		return a.executeSSHCommand(toolCall.Input)
	default:
		return fmt.Sprintf("未知工具: %s", toolCall.Tool), nil
	}
}

// executeMySQLQuery 执行MySQL查询
func (a *Agent) executeMySQLQuery(sql string) (string, error) {
	result, err := a.mysqlTool.Query(sql)
	if err != nil {
		return "", err
	}

	if len(result) == 0 {
		return "查询执行成功，无结果返回。", nil
	}

	// 格式化结果为表格
	var lines []string

	// 获取列名
	headers := make([]string, 0, len(result[0]))
	for h := range result[0] {
		headers = append(headers, h)
	}

	// 表头
	lines = append(lines, strings.Join(headers, " | "))

	// 分隔线
	sepParts := make([]string, len(headers))
	for i, h := range headers {
		sepParts[i] = strings.Repeat("-", len(h))
	}
	lines = append(lines, strings.Join(sepParts, "-+-"))

	// 数据行
	for _, row := range result {
		rowParts := make([]string, len(headers))
		for i, h := range headers {
			val := row[h]
			if val == nil {
				rowParts[i] = "NULL"
			} else {
				rowParts[i] = fmt.Sprintf("%v", val)
			}
		}
		lines = append(lines, strings.Join(rowParts, " | "))
	}

	return strings.Join(lines, "\n"), nil
}

// executeSSHCommand 执行SSH命令
func (a *Agent) executeSSHCommand(command string) (string, error) {
	return a.sshTool.Execute(command, 60)
}
