// Package mysql 提供MySQL查询执行功能
//
// 该包提供安全的MySQL连接管理和只读查询执行功能,
// 包含SQL白名单验证机制,确保只执行安全的只读查询。
// 支持一次执行多条用分号分隔的查询。
package mysql

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Client MySQL客户端
//
// Client 负责管理MySQL连接和执行只读查询。
// 所有SQL都会经过白名单验证,确保只执行安全的只读操作。
type Client struct {
	host      string
	port      int
	username  string
	password  string
	database  *string
	db        *sql.DB
	validator *Validator
	logger    *slog.Logger
}

// NewClient 创建一个新的MySQL客户端
//
// 参数:
//   - host: MySQL主机地址
//   - port: MySQL端口
//   - username: MySQL用户名
//   - password: MySQL密码
//   - database: MySQL数据库名(可选)
//
// 返回: 新的MySQL客户端实例
func NewClient(host string, port int, username, password string, database *string) *Client {
	return &Client{
		host:      host,
		port:      port,
		username:  username,
		password:  password,
		database:  database,
		validator: NewValidator(),
		logger:    slog.Default(),
	}
}

// Connect 建立MySQL连接
//
// 如果已连接则不做任何操作。
func (c *Client) Connect() error {
	if c.db != nil {
		return nil
	}

	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/",
		c.username, c.password, c.host, c.port)
	if c.database != nil && *c.database != "" {
		dsn += *c.database
	}
	dsn += "?charset=utf8mb4&parseTime=True&loc=Local"

	c.logger.Info("正在连接MySQL", "addr", fmt.Sprintf("%s:%d", c.host, c.port), "user", c.username)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("创建MySQL连接失败: %w", err)
	}

	// 设置连接参数
	db.SetConnMaxLifetime(1 * time.Hour)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("MySQL连接失败: %w", err)
	}

	c.db = db
	c.logger.Info("MySQL连接成功")
	return nil
}

// Disconnect 关闭MySQL连接
func (c *Client) Disconnect() {
	if c.db != nil {
		c.db.Close()
		c.db = nil
		c.logger.Info("MySQL连接已断开")
	}
}

// Query 执行只读MySQL查询
//
// 支持一次执行多条用分号分隔的查询。
// 多条查询时不支持参数化查询。
//
// 参数:
//   - sql: SQL查询语句
//   - params: 查询参数(可选,仅单条查询时有效)
//
// 返回: 查询结果,每行是一个map[string]any,或错误信息
func (c *Client) Query(sqlStr string, params ...any) ([]map[string]any, error) {
	// 先去除MySQL命令行语法 \G
	sqlStr = strings.TrimSuffix(strings.TrimSpace(sqlStr), "\\G")

	queries := SplitMultiQuery(sqlStr)
	if len(queries) == 0 {
		return []map[string]any{}, nil
	}

	// 多条查询时不支持参数
	if len(queries) > 1 {
		if len(params) > 0 {
			c.logger.Warn("多条查询不支持参数,参数将被忽略")
		}
		var allResults []map[string]any
		for i, query := range queries {
			c.logger.Info("执行多查询", "index", i+1, "total", len(queries), "sql", query)
			results, err := c.executeSingleQuery(query)
			if err != nil {
				return nil, fmt.Errorf("第%d条查询执行失败: %w", i+1, err)
			}
			allResults = append(allResults, results...)
		}
		return allResults, nil
	}

	return c.executeSingleQuery(queries[0], params...)
}

// executeSingleQuery 执行单条查询(内部方法)
func (c *Client) executeSingleQuery(sqlStr string, params ...any) ([]map[string]any, error) {
	// 验证查询
	valid, reason := c.validator.Validate(sqlStr)
	if !valid {
		return nil, fmt.Errorf("查询验证失败: %s", reason)
	}

	// 确保已连接
	if c.db == nil {
		if err := c.Connect(); err != nil {
			return nil, err
		}
	}

	c.logger.Info("执行MySQL查询", "sql", sqlStr)

	rows, err := c.db.Query(sqlStr, params...)
	if err != nil {
		return nil, fmt.Errorf("查询执行失败: %w", err)
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列名失败: %w", err)
	}

	var results []map[string]any

	for rows.Next() {
		// 创建接收变量
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("扫描行失败: %w", err)
		}

		// 构建结果map
		row := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			// 处理[]byte为string
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果失败: %w", err)
	}

	return results, nil
}

// QueryOne 执行查询并返回单行结果
//
// 如果有多行结果,只返回第一行。
func (c *Client) QueryOne(sqlStr string, params ...any) (map[string]any, error) {
	results, err := c.Query(sqlStr, params...)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

// QueryValue 执行查询并返回第一列的单个值
func (c *Client) QueryValue(sqlStr string, params ...any) (any, error) {
	row, err := c.QueryOne(sqlStr, params...)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	// 返回第一个值
	for _, v := range row {
		return v, nil
	}
	return nil, nil
}

// ShowVariables 获取MySQL系统变量
//
// 参数:
//   - like: LIKE模式(可选,为空则获取所有变量)
//
// 返回: 变量名到值的map
func (c *Client) ShowVariables(like string) (map[string]string, error) {
	var sqlStr string
	var params []any

	if like != "" {
		sqlStr = "SHOW VARIABLES LIKE ?"
		params = []any{like}
	} else {
		sqlStr = "SHOW VARIABLES"
	}

	rows, err := c.Query(sqlStr, params...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, row := range rows {
		varName, _ := row["Variable_name"].(string)
		varValue, _ := row["Value"].(string)
		result[varName] = varValue
	}
	return result, nil
}

// ShowStatus 获取MySQL状态信息
//
// 参数:
//   - like: LIKE模式(可选,为空则获取所有状态)
//
// 返回: 状态名到值的map
func (c *Client) ShowStatus(like string) (map[string]string, error) {
	var sqlStr string
	var params []any

	if like != "" {
		sqlStr = "SHOW GLOBAL STATUS LIKE ?"
		params = []any{like}
	} else {
		sqlStr = "SHOW GLOBAL STATUS"
	}

	rows, err := c.Query(sqlStr, params...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, row := range rows {
		varName, _ := row["Variable_name"].(string)
		varValue, _ := row["Value"].(string)
		result[varName] = varValue
	}
	return result, nil
}

// ShowProcesslist 获取当前进程列表
func (c *Client) ShowProcesslist() ([]map[string]any, error) {
	return c.Query("SHOW FULL PROCESSLIST")
}

// ShowEngineStatus 获取InnoDB引擎状态
func (c *Client) ShowEngineStatus() (map[string]any, error) {
	rows, err := c.Query("SHOW ENGINE INNODB STATUS")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// SetLogger 设置日志记录器
func (c *Client) SetLogger(logger *slog.Logger) {
	c.logger = logger
}
