package config

import (
	"os"
	"strconv"
)

// getEnvRequired 获取必需的环境变量
//
// 如果环境变量不存在或为空,返回空字符串
func getEnvRequired(name string) string {
	return os.Getenv(name)
}

// getEnvWithDefault 获取环境变量,如果不存在则返回默认值
func getEnvWithDefault(name, defaultValue string) string {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvPtr 获取环境变量指针
//
// 如果环境变量不存在,返回nil;否则返回指向该值的指针
func getEnvPtr(name string) *string {
	value := os.Getenv(name)
	if value == "" {
		return nil
	}
	return &value
}

// getEnvAsInt 获取环境变量并转换为int类型
//
// 如果环境变量不存在或解析失败,返回默认值和错误
func getEnvAsInt(name string, defaultValue int) (int, error) {
	valueStr := os.Getenv(name)
	if valueStr == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue, err
	}
	return value, nil
}

// getEnvAsFloat 获取环境变量并转换为float64类型
//
// 如果环境变量不存在或解析失败,返回默认值和错误
func getEnvAsFloat(name string, defaultValue float64) (float64, error) {
	valueStr := os.Getenv(name)
	if valueStr == "" {
		return defaultValue, nil
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue, err
	}
	return value, nil
}
