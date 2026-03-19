// Package toolcall 定义工具调用数据结构
//
// 该包提供ToolCall数据结构，用于表示LLM返回的工具调用决策。
package toolcall

// ToolCall 表示一个工具调用决策
//
// ToolCall包含LLM决策使用哪个工具、执行什么动作以及输入数据。
type ToolCall struct {
	Tool     string `json:"tool"`     // 工具名称: mysql_query, ssh_exec, final_answer
	Action   string `json:"action"`   // 动作描述
	Input    string `json:"input"`    // 输入数据: SQL查询或Linux命令
}

// NewToolCall 创建一个新的ToolCall实例
func NewToolCall(tool, action, input string) *ToolCall {
	return &ToolCall{
		Tool:   tool,
		Action: action,
		Input:  input,
	}
}

// ToMap 将ToolCall转换为map格式
func (tc *ToolCall) ToMap() map[string]string {
	return map[string]string{
		"tool":   tc.Tool,
		"action": tc.Action,
		"input":  tc.Input,
	}
}

// IsFinalAnswer 检查是否是最终答案
func (tc *ToolCall) IsFinalAnswer() bool {
	return tc.Tool == "final_answer"
}

// IsMySQLQuery 检查是否是MySQL查询
func (tc *ToolCall) IsMySQLQuery() bool {
	return tc.Tool == "mysql_query"
}

// IsSSHExec 检查是否是SSH命令执行
func (tc *ToolCall) IsSSHExec() bool {
	return tc.Tool == "ssh_exec"
}
