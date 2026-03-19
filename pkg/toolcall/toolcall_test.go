package toolcall

import "testing"

// TestNewToolCall 测试创建ToolCall
func TestNewToolCall(t *testing.T) {
	tc := NewToolCall("mysql_query", "查询状态", "SHOW STATUS")

	if tc.Tool != "mysql_query" {
		t.Errorf("Tool = %q, want %q", tc.Tool, "mysql_query")
	}
	if tc.Action != "查询状态" {
		t.Errorf("Action = %q, want %q", tc.Action, "查询状态")
	}
	if tc.Input != "SHOW STATUS" {
		t.Errorf("Input = %q, want %q", tc.Input, "SHOW STATUS")
	}
}

// TestToolCall_ToMap 测试ToMap方法
func TestToolCall_ToMap(t *testing.T) {
	tc := &ToolCall{
		Tool:   "ssh_exec",
		Action: "执行命令",
		Input:  "uptime",
	}

	m := tc.ToMap()

	if m["tool"] != "ssh_exec" {
		t.Errorf("map['tool'] = %q, want %q", m["tool"], "ssh_exec")
	}
	if m["action"] != "执行命令" {
		t.Errorf("map['action'] = %q, want %q", m["action"], "执行命令")
	}
	if m["input"] != "uptime" {
		t.Errorf("map['input'] = %q, want %q", m["input"], "uptime")
	}
}

// TestToolCall_TypeChecks 测试类型检查方法
func TestToolCall_TypeChecks(t *testing.T) {
	tests := []struct {
		name        string
		tool        string
		isFinal     bool
		isMySQL     bool
		isSSH       bool
	}{
		{
			name:        "final_answer",
			tool:        "final_answer",
			isFinal:     true,
			isMySQL:     false,
			isSSH:       false,
		},
		{
			name:        "mysql_query",
			tool:        "mysql_query",
			isFinal:     false,
			isMySQL:     true,
			isSSH:       false,
		},
		{
			name:        "ssh_exec",
			tool:        "ssh_exec",
			isFinal:     false,
			isMySQL:     false,
			isSSH:       true,
		},
		{
			name:        "unknown",
			tool:        "unknown",
			isFinal:     false,
			isMySQL:     false,
			isSSH:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &ToolCall{Tool: tt.tool}
			if tc.IsFinalAnswer() != tt.isFinal {
				t.Errorf("IsFinalAnswer() = %v, want %v", tc.IsFinalAnswer(), tt.isFinal)
			}
			if tc.IsMySQLQuery() != tt.isMySQL {
				t.Errorf("IsMySQLQuery() = %v, want %v", tc.IsMySQLQuery(), tt.isMySQL)
			}
			if tc.IsSSHExec() != tt.isSSH {
				t.Errorf("IsSSHExec() = %v, want %v", tc.IsSSHExec(), tt.isSSH)
			}
		})
	}
}
