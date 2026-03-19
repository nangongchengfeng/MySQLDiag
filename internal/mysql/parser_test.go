package mysql

import "testing"

// TestSplitMultiQuery_SingleQuery 测试单条查询
func TestSplitMultiQuery_SingleQuery(t *testing.T) {
	sql := "SELECT * FROM users"
	queries := SplitMultiQuery(sql)

	if len(queries) != 1 {
		t.Errorf("期望1条查询,实际%d条", len(queries))
	}
	if queries[0] != sql {
		t.Errorf("期望查询 %q,实际 %q", sql, queries[0])
	}
}

// TestSplitMultiQuery_MultipleQueries 测试多条查询
func TestSplitMultiQuery_MultipleQueries(t *testing.T) {
	sql := "SELECT * FROM users; SHOW TABLES; SELECT * FROM posts"
	queries := SplitMultiQuery(sql)

	if len(queries) != 3 {
		t.Errorf("期望3条查询,实际%d条", len(queries))
	}
	if queries[0] != "SELECT * FROM users" {
		t.Errorf("第1条查询不匹配")
	}
	if queries[1] != "SHOW TABLES" {
		t.Errorf("第2条查询不匹配")
	}
	if queries[2] != "SELECT * FROM posts" {
		t.Errorf("第3条查询不匹配")
	}
}

// TestSplitMultiQuery_StringWithSemicolon 测试字符串中包含分号
func TestSplitMultiQuery_StringWithSemicolon(t *testing.T) {
	sql := "SELECT * FROM users WHERE name = 'test;123'"
	queries := SplitMultiQuery(sql)

	if len(queries) != 1 {
		t.Errorf("期望1条查询,实际%d条", len(queries))
	}
	if queries[0] != sql {
		t.Errorf("查询不应该被拆分")
	}
}

// TestSplitMultiQuery_DoubleQuotedString 测试双引号字符串
func TestSplitMultiQuery_DoubleQuotedString(t *testing.T) {
	sql := `SELECT * FROM users WHERE name = "test;123"`
	queries := SplitMultiQuery(sql)

	if len(queries) != 1 {
		t.Errorf("期望1条查询,实际%d条", len(queries))
	}
}

// TestSplitMultiQuery_EscapedCharacters 测试转义字符
func TestSplitMultiQuery_EscapedCharacters(t *testing.T) {
	sql := `SELECT * FROM users WHERE name = 'test\';123'`
	queries := SplitMultiQuery(sql)

	if len(queries) != 1 {
		t.Errorf("期望1条查询,实际%d条", len(queries))
	}
}

// TestSplitMultiQuery_Whitespace 测试空白处理
func TestSplitMultiQuery_Whitespace(t *testing.T) {
	sql := "  SELECT * FROM users  ;  SHOW TABLES  "
	queries := SplitMultiQuery(sql)

	if len(queries) != 2 {
		t.Errorf("期望2条查询,实际%d条", len(queries))
	}
	if queries[0] != "SELECT * FROM users" {
		t.Errorf("第1条查询应该去除首尾空白")
	}
	if queries[1] != "SHOW TABLES" {
		t.Errorf("第2条查询应该去除首尾空白")
	}
}

// TestSplitMultiQuery_EmptyQueries 测试空查询
func TestSplitMultiQuery_EmptyQueries(t *testing.T) {
	sql := "SELECT * FROM users;; SHOW TABLES;;"
	queries := SplitMultiQuery(sql)

	if len(queries) != 2 {
		t.Errorf("期望2条查询,实际%d条", len(queries))
	}
}

// TestTrimSpace 测试trimSpace函数
func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  test  ", "test"},
		{"\ttest\n", "test"},
		{"test", "test"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		result := trimSpace(tt.input)
		if result != tt.expected {
			t.Errorf("trimSpace(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
