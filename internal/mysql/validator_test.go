package mysql

import "testing"

// TestValidator_Validate_AllowedQueries 测试允许的查询
func TestValidator_Validate_AllowedQueries(t *testing.T) {
	v := NewValidator()

	allowedQueries := []string{
		"SELECT * FROM users",
		"SHOW TABLES",
		"SHOW VARIABLES",
		"SHOW GLOBAL STATUS",
		"DESCRIBE users",
		"DESC users",
		"EXPLAIN SELECT * FROM users",
		"USE testdb",
		"ANALYZE TABLE users",
		"CHECK TABLE users",
	}

	for _, q := range allowedQueries {
		valid, reason := v.Validate(q)
		if !valid {
			t.Errorf("查询 %q 应该被允许,但被拒绝: %s", q, reason)
		}
	}
}

// TestValidator_Validate_BlockedQueries 测试禁止的查询
func TestValidator_Validate_BlockedQueries(t *testing.T) {
	v := NewValidator()

	blockedQueries := []string{
		"INSERT INTO users VALUES (1, 'test')",
		"UPDATE users SET name = 'test' WHERE id = 1",
		"DELETE FROM users WHERE id = 1",
		"REPLACE INTO users VALUES (1, 'test')",
		"DROP TABLE users",
		"TRUNCATE TABLE users",
		"ALTER TABLE users ADD COLUMN age INT",
		"CREATE TABLE users (id INT)",
		"RENAME TABLE users TO users_old",
		"GRANT ALL ON *.* TO 'user'@'%'",
		"REVOKE ALL ON *.* FROM 'user'@'%'",
		"SET GLOBAL max_connections = 1000",
		"LOAD DATA INFILE '/tmp/data.txt' INTO TABLE users",
		"LOCK TABLES users WRITE",
		"UNLOCK TABLES",
		"START TRANSACTION",
		"COMMIT",
		"ROLLBACK",
		"FLUSH PRIVILEGES",
		"RESET QUERY CACHE",
		"SHUTDOWN",
		"KILL 123",
		"PURGE BINARY LOGS TO 'mysql-bin.000100'",
		"OPTIMIZE TABLE users",
		"REPAIR TABLE users",
	}

	for _, q := range blockedQueries {
		valid, _ := v.Validate(q)
		if valid {
			t.Errorf("查询 %q 应该被拒绝,但被允许", q)
		}
	}
}

// TestValidator_Validate_Comments 测试带注释的查询
func TestValidator_Validate_Comments(t *testing.T) {
	v := NewValidator()

	// 带--注释的查询
	valid, reason := v.Validate("SELECT * FROM users -- this is a comment")
	if !valid {
		t.Errorf("带--注释的查询应该被允许: %s", reason)
	}

	// 带#注释的查询
	valid, reason = v.Validate("SHOW TABLES # this is a comment")
	if !valid {
		t.Errorf("带#注释的查询应该被允许: %s", reason)
	}

	// 多行带注释的查询
	multiLine := `SELECT *
FROM users
-- where id = 1
WHERE name = 'test'`
	valid, reason = v.Validate(multiLine)
	if !valid {
		t.Errorf("多行带注释的查询应该被允许: %s", reason)
	}
}

// TestValidator_Validate_EmptyQuery 测试空查询
func TestValidator_Validate_EmptyQuery(t *testing.T) {
	v := NewValidator()
	valid, reason := v.Validate("")
	if valid {
		t.Error("空查询应该被拒绝")
	}
	if reason != "空查询" {
		t.Errorf("空查询的原因应该是'空查询',实际是: %s", reason)
	}
}

// TestValidator_Validate_UnknownStatement 测试未知语句
func TestValidator_Validate_UnknownStatement(t *testing.T) {
	v := NewValidator()
	valid, _ := v.Validate("SOME_UNKNOWN_STATEMENT")
	if valid {
		t.Error("未知语句应该被拒绝")
	}
}
