package ssh

import "testing"

// TestValidator_Validate_AllowedCommands 测试允许的命令
func TestValidator_Validate_AllowedCommands(t *testing.T) {
	v := NewValidator()

	allowedCommands := []string{
		"uptime",
		"ls -la",
		"cat /proc/cpuinfo",
		"free -h",
		"df -h",
		"ps aux",
		"netstat -tuln",
		"mysqladmin ping",
		"mysqladmin status",
		"mysqldump --no-data testdb",
		"mysqldump -d testdb",
	}

	for _, cmd := range allowedCommands {
		valid, reason := v.Validate(cmd)
		if !valid {
			t.Errorf("命令 %q 应该被允许,但被拒绝: %s", cmd, reason)
		}
	}
}

// TestValidator_Validate_BlockedCommands 测试禁止的命令
func TestValidator_Validate_BlockedCommands(t *testing.T) {
	v := NewValidator()

	blockedCommands := []string{
		"rm -rf /",
		"echo hello > file.txt",
		"cat file >> another.txt",
		"ls && rm file",
		"ls || echo failed",
		"rm `ls`",
		"rm $(ls)",
		"sudo ls",
		"su root",
		"wget http://example.com",
		"curl http://example.com",
		"git status",
		"docker ps",
		":(){ :|:& };:",
	}

	for _, cmd := range blockedCommands {
		valid, _ := v.Validate(cmd)
		if valid {
			t.Errorf("命令 %q 应该被拒绝,但被允许", cmd)
		}
	}
}

// TestValidator_Validate_MySQLAdmin 测试mysqladmin命令
func TestValidator_Validate_MySQLAdmin(t *testing.T) {
	v := NewValidator()

	// 允许的子命令
	allowed := []string{
		"mysqladmin ping",
		"mysqladmin status",
		"mysqladmin version",
		"mysqladmin extended-status",
		"mysqladmin processlist",
		"mysqladmin variables",
	}

	for _, cmd := range allowed {
		valid, reason := v.Validate(cmd)
		if !valid {
			t.Errorf("命令 %q 应该被允许,但被拒绝: %s", cmd, reason)
		}
	}

	// 禁止的子命令
	blocked := []string{
		"mysqladmin shutdown",
		"mysqladmin kill 123",
		"mysqladmin drop testdb",
	}

	for _, cmd := range blocked {
		valid, _ := v.Validate(cmd)
		if valid {
			t.Errorf("命令 %q 应该被拒绝,但被允许", cmd)
		}
	}
}

// TestValidator_Validate_MySQLDump 测试mysqldump命令
func TestValidator_Validate_MySQLDump(t *testing.T) {
	v := NewValidator()

	// 允许的(仅结构)
	allowed := []string{
		"mysqldump --no-data testdb",
		"mysqldump -d testdb",
		"mysqldump --no-data --single-transaction testdb",
		"mysqldump --no-data --databases testdb",
	}

	for _, cmd := range allowed {
		valid, reason := v.Validate(cmd)
		if !valid {
			t.Errorf("命令 %q 应该被允许,但被拒绝: %s", cmd, reason)
		}
	}

	// 禁止的(包含数据)
	blocked := []string{
		"mysqldump testdb",
	}

	for _, cmd := range blocked {
		valid, _ := v.Validate(cmd)
		if valid {
			t.Errorf("命令 %q 应该被拒绝,但被允许", cmd)
		}
	}
}

// TestValidator_Validate_EmptyCommand 测试空命令
func TestValidator_Validate_EmptyCommand(t *testing.T) {
	v := NewValidator()
	valid, reason := v.Validate("")
	if valid {
		t.Error("空命令应该被拒绝")
	}
	if reason != "空命令" {
		t.Errorf("空命令的原因应该是'空命令',实际是: %s", reason)
	}
}

// TestValidator_Validate_UnknownCommand 测试未知命令
func TestValidator_Validate_UnknownCommand(t *testing.T) {
	v := NewValidator()
	valid, _ := v.Validate("some_unknown_command")
	if valid {
		t.Error("未知命令应该被拒绝")
	}
}
