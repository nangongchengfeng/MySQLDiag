package config

import (
	"os"
	"testing"
)

// TestConfig_AddrMethods 测试地址方法
func TestConfig_AddrMethods(t *testing.T) {
	cfg := &Config{
		SSHHost: "test.example.com",
		SSHPort: 2222,
		MySQLHost: "db.example.com",
		MySQLPort: 3307,
	}

	if cfg.SSHAddr() != "test.example.com:2222" {
		t.Errorf("SSHAddr() = %q, want %q", cfg.SSHAddr(), "test.example.com:2222")
	}

	if cfg.MySQLAddr() != "db.example.com:3307" {
		t.Errorf("MySQLAddr() = %q, want %q", cfg.MySQLAddr(), "db.example.com:3307")
	}
}

// TestConfig_MySQLDSN 测试MySQL DSN生成
func TestConfig_MySQLDSN(t *testing.T) {
	// 测试无数据库名的情况
	cfg1 := &Config{
		MySQLUser: "user",
		MySQLPassword: "pass",
		MySQLHost: "localhost",
		MySQLPort: 3306,
	}
	expected1 := "user:pass@tcp(localhost:3306)/?charset=utf8mb4&parseTime=True&loc=Local"
	if cfg1.MySQLDSN() != expected1 {
		t.Errorf("MySQLDSN() = %q, want %q", cfg1.MySQLDSN(), expected1)
	}

	// 测试有数据库名的情况
	dbName := "testdb"
	cfg2 := &Config{
		MySQLUser: "user",
		MySQLPassword: "pass",
		MySQLHost: "localhost",
		MySQLPort: 3306,
		MySQLDatabase: &dbName,
	}
	expected2 := "user:pass@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	if cfg2.MySQLDSN() != expected2 {
		t.Errorf("MySQLDSN() = %q, want %q", cfg2.MySQLDSN(), expected2)
	}
}

// TestEnvHelpers 测试环境变量辅助函数
func TestEnvHelpers(t *testing.T) {
	// 测试 getEnvWithDefault
	os.Setenv("TEST_ENV_NOT_EXIST", "")
	if getEnvWithDefault("TEST_ENV_NOT_EXIST", "default") != "default" {
		t.Error("getEnvWithDefault should return default for non-existent env")
	}

	os.Setenv("TEST_ENV_EXIST", "value")
	if getEnvWithDefault("TEST_ENV_EXIST", "default") != "value" {
		t.Error("getEnvWithDefault should return env value when exists")
	}

	// 测试 getEnvPtr
	os.Setenv("TEST_ENV_PTR", "ptr_value")
	ptr := getEnvPtr("TEST_ENV_PTR")
	if ptr == nil || *ptr != "ptr_value" {
		t.Error("getEnvPtr should return pointer to value")
	}

	os.Setenv("TEST_ENV_PTR_EMPTY", "")
	ptr = getEnvPtr("TEST_ENV_PTR_EMPTY")
	if ptr != nil {
		t.Error("getEnvPtr should return nil for empty value")
	}

	// 测试 getEnvAsInt
	os.Setenv("TEST_ENV_INT", "123")
	val, err := getEnvAsInt("TEST_ENV_INT", 456)
	if err != nil || val != 123 {
		t.Errorf("getEnvAsInt failed: err=%v, val=%d", err, val)
	}

	os.Setenv("TEST_ENV_INT_INVALID", "abc")
	val, err = getEnvAsInt("TEST_ENV_INT_INVALID", 456)
	if err == nil {
		t.Error("getEnvAsInt should return error for invalid int")
	}
	if val != 456 {
		t.Error("getEnvAsInt should return default on error")
	}

	// 测试 getEnvAsFloat
	os.Setenv("TEST_ENV_FLOAT", "123.45")
	fval, err := getEnvAsFloat("TEST_ENV_FLOAT", 67.89)
	if err != nil || fval != 123.45 {
		t.Errorf("getEnvAsFloat failed: err=%v, val=%f", err, fval)
	}

	os.Unsetenv("TEST_ENV_NOT_EXIST")
	os.Unsetenv("TEST_ENV_EXIST")
	os.Unsetenv("TEST_ENV_PTR")
	os.Unsetenv("TEST_ENV_PTR_EMPTY")
	os.Unsetenv("TEST_ENV_INT")
	os.Unsetenv("TEST_ENV_INT_INVALID")
	os.Unsetenv("TEST_ENV_FLOAT")
}
