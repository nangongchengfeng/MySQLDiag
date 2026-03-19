package report

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewWriter 测试创建Writer
func TestNewWriter(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "report-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(tmpDir)
	if err != nil {
		t.Fatalf("NewWriter失败: %v", err)
	}

	if writer == nil {
		t.Error("Writer不应该为nil")
	}

	if writer.outputDir != tmpDir {
		t.Errorf("outputDir = %q, want %q", writer.outputDir, tmpDir)
	}
}

// TestWriter_AddObservation 测试添加观察
func TestWriter_AddObservation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "report-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, _ := NewWriter(tmpDir)

	writer.AddObservation("mysql_query", "查询状态", "SHOW STATUS", "结果...", 1)

	obs := writer.Observations()
	if len(obs) != 1 {
		t.Errorf("期望1条观察,实际%d条", len(obs))
	}
	if obs[0].Tool != "mysql_query" {
		t.Errorf("Tool = %q, want %q", obs[0].Tool, "mysql_query")
	}
}

// TestWriter_GenerateMarkdown 测试生成Markdown
func TestWriter_GenerateMarkdown(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "report-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, _ := NewWriter(tmpDir)
	writer.AddObservation("mysql_query", "查询", "SHOW TABLES", "结果", 1)
	writer.AddObservation("ssh_exec", "执行", "uptime", "00:00:00 up 1 day", 1)

	metadata := map[string]any{
		"目标SSH": "test.example.com:22",
		"目标MySQL": "db.example.com:3306",
	}

	path, err := writer.GenerateMarkdown("这是最终诊断", metadata)
	if err != nil {
		t.Fatalf("GenerateMarkdown失败: %v", err)
	}

	if !filepath.IsAbs(path) {
		t.Errorf("路径应该是绝对路径: %s", path)
	}

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("报告文件不存在: %s", path)
	}

	// 读取并检查内容
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取报告失败: %v", err)
	}

	if len(content) == 0 {
		t.Error("报告内容不应该为空")
	}
}

// TestWriter_GenerateSummaryText 测试生成摘要
func TestWriter_GenerateSummaryText(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "report-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, _ := NewWriter(tmpDir)
	writer.AddObservation("mysql_query", "查询", "SHOW TABLES", "结果", 1)

	summary := writer.GenerateSummaryText("最终诊断结果")
	if summary == "" {
		t.Error("摘要不应该为空")
	}
}

// TestNewObservation 测试创建Observation
func TestNewObservation(t *testing.T) {
	before := time.Now()
	obs := NewObservation(1, "tool", "action", "input", "observation")
	after := time.Now()

	if obs.Round != 1 {
		t.Errorf("Round = %d, want %d", obs.Round, 1)
	}
	if obs.Tool != "tool" {
		t.Errorf("Tool = %q, want %q", obs.Tool, "tool")
	}
	if obs.Timestamp.Before(before) || obs.Timestamp.After(after) {
		t.Error("Timestamp不在预期范围内")
	}
}

// TestFormatCodeBlock 测试formatCodeBlock
func TestFormatCodeBlock(t *testing.T) {
	// 测试空内容
	if formatCodeBlock("", "") != "" {
		t.Error("空内容应该返回空")
	}

	// 测试普通内容
	result := formatCodeBlock("test content", "sql")
	expected := "```sql\ntest content\n```"
	if result != expected {
		t.Errorf("formatCodeBlock = %q, want %q", result, expected)
	}

	// 测试长内容截断
	longContent := make([]byte, 6000)
	for i := range longContent {
		longContent[i] = 'x'
	}
	result = formatCodeBlock(string(longContent), "")
	if len(result) > 5100 { // 5000 + 一些额外字符
		t.Error("长内容应该被截断")
	}
}

// TestGetMaxRound 测试getMaxRound
func TestGetMaxRound(t *testing.T) {
	tests := []struct {
		name     string
		obs      []Observation
		expected int
	}{
		{
			name:     "空切片",
			obs:      []Observation{},
			expected: 0,
		},
		{
			name: "单条观察",
			obs: []Observation{{Round: 1}},
			expected: 1,
		},
		{
			name: "多条观察",
			obs: []Observation{
				{Round: 1},
				{Round: 2},
				{Round: 3},
			},
			expected: 3,
		},
		{
			name: "乱序观察",
			obs: []Observation{
				{Round: 3},
				{Round: 1},
				{Round: 2},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMaxRound(tt.obs)
			if result != tt.expected {
				t.Errorf("getMaxRound = %d, want %d", result, tt.expected)
			}
		})
	}
}
