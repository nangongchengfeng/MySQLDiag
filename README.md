# MySQL/Linux 远程只读诊断 Agent

这是一个使用LLM自主决策的MySQL和Linux系统性能诊断工具，能够自动收集信息并生成综合诊断报告。

## 项目概述

该诊断Agent具有以下特点：

- **自主诊断**: 使用LLM自动决定需要收集哪些信息
- **只读安全**: 严格的白名单机制，确保只执行只读操作
- **双语言支持**: 提供Python原版和Go重构版本
- **中文界面**: 所有提示词、注释和报告均使用中文
- **Markdown报告**: 生成详细的Markdown格式诊断报告

## 快速开始

### Python版本

```bash
# 安装依赖
uv sync

# 运行
uv run python main.py
```

### Go版本

```bash
# 编译
go build -o bin/agent.exe ./cmd/agent

# 运行
bin/agent.exe

# 或直接运行
go run ./cmd/agent
```

## 配置

复制 `.env.example` 为 `.env` 并填写配置：

```bash
cp .env.example .env
```

### 配置项说明

| 配置项 | 说明 | 必填 |
|--------|------|------|
| SILICONFLOW_API_KEY | SiliconFlow API密钥 | 是 |
| SILICONFLOW_BASE_URL | API基础URL | 否，默认https://api.siliconflow.cn/v1/ |
| SILICONFLOW_MODEL | 使用的模型 | 否，默认deepseek-ai/DeepSeek-V3.2 |
| SSH_HOST | SSH主机地址 | 是 |
| SSH_PORT | SSH端口 | 否，默认22 |
| SSH_USER | SSH用户名 | 否，默认root |
| SSH_PASSWORD | SSH密码 | 可选 |
| SSH_KEY_PATH | SSH密钥路径 | 可选 |
| MYSQL_HOST | MySQL主机地址 | 是 |
| MYSQL_PORT | MySQL端口 | 否，默认3306 |
| MYSQL_USER | MySQL用户名 | 否，默认root |
| MYSQL_PASSWORD | MySQL密码 | 否，默认空 |
| MYSQL_DATABASE | MySQL数据库名 | 可选 |
| SLOW_QUERY_THRESHOLD | 慢查询阈值(秒) | 否，默认0.5 |

## 项目结构

### Python版本

```
agent/
├── main.py                              # 主入口
├── src/mysql_diagnostic_agent/
│   ├── config.py                        # 配置管理
│   ├── tools/
│   │   ├── ssh_tool.py                  # SSH工具
│   │   └── mysql_tool.py                # MySQL工具
│   ├── agent/
│   │   └── diagnostic_agent.py          # 核心Agent
│   ├── report/
│   │   └── report_writer.py             # 报告生成
│   └── core/
│       └── tool_call.py                 # 工具调用结构
```

### Go版本

```
agent/
├── cmd/agent/
│   └── main.go                          # 主入口
├── internal/
│   ├── config/                          # 配置管理
│   ├── ssh/                             # SSH工具
│   ├── mysql/                           # MySQL工具
│   ├── agent/                           # 核心Agent
│   └── report/                          # 报告生成
├── pkg/toolcall/                        # 工具调用结构
├── go.mod
└── go.sum
```

## 安全特性

### SSH命令白名单

只允许执行以下类别的只读命令：

- 系统信息: top, uptime, uname, hostname, date, ls, cat
- 资源监控: free, df, du, vmstat, iostat, mpstat, sar
- 进程管理: ps, pstree, pgrep, pidof, lsof, fuser
- 网络工具: netstat, ss, ip, ifconfig, ping, traceroute, nslookup, dig
- 系统日志: dmesg, journalctl
- 系统状态: lscpu, lsmem, lsblk
- MySQL相关: mysqladmin(只读), mysqldump(--no-data)
- 文本处理: grep, awk, sed, cut, sort, uniq, wc, head, tail
- 性能监控: iotop, htop, atop
- 服务管理: systemctl

### MySQL查询白名单

只允许执行以下类型的只读查询：

- SELECT
- SHOW
- DESCRIBE / DESC
- EXPLAIN
- USE
- HELP
- CHECKSUM
- CHECK
- ANALYZE

## 可用工具

### 1. mysql_query
执行只读MySQL查询
- 输入: SQL查询语句
- 用途: 检查MySQL状态、变量、进程列表、慢查询、锁信息等

### 2. ssh_exec
通过SSH执行只读Linux命令
- 输入: Linux命令
- 用途: 检查系统资源：CPU、内存、磁盘、网络、进程等

### 3. final_answer
结束诊断并给出结论
- 输入: 综合诊断报告和优化建议
- 使用时机: 当收集了足够的信息，可以给出完整诊断时

## 诊断流程

### 第1阶段 - 基础检查（第1-3轮）
1. 慢查询统计 - Slow_queries, long_query_time
2. MySQL运行时间 - Uptime
3. 查询量统计 - Questions, 计算QPS
4. 连接数统计 - Threads_connected, Max_used_connections
5. 系统基本信息 - uptime, free, df

### 第2阶段 - 深入检查（第4-10轮）
1. 进程列表 - SHOW FULL PROCESSLIST
2. InnoDB状态 - SHOW ENGINE INNODB STATUS
3. 锁等待信息
4. CPU使用率 - top, vmstat
5. 内存使用 - free, ps
6. 磁盘I/O - iostat, df
7. 网络连接 - netstat, ss

### 第3阶段 - 针对性分析（按需进行）
1. 表缓存状态
2. 查询缓存（如果启用）
3. 临时表使用情况
4. 排序统计
5. 特定慢查询的EXPLAIN

## 开发指南

### 运行测试

**Python版本:**
```bash
.venv/Scripts/python.exe -m py_compile src/mysql_diagnostic_agent/*.py main.py
```

**Go版本:**
```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/config -v
go test ./internal/ssh -v
go test ./internal/mysql -v
go test ./internal/report -v
go test ./pkg/toolcall -v
```

### 代码规范

- Python: 遵循PEP 8规范
- Go: 遵循Go官方代码规范
- 所有代码都有完整的中文注释

## 最终诊断报告内容

当使用final_answer时，报告会包含以下内容：

1. 总体评价 - 健康状态评分（0-100分）
2. 慢查询分析 - 数量、阈值、占比
3. QPS和负载分析
4. 连接和线程状态
5. 锁等待分析
6. 内存使用状况
7. CPU使用状况
8. 磁盘空间和I/O
9. 主要问题列表（按严重程度排序）
10. 具体的优化建议
