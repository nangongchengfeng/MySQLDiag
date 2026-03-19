// main.go MySQL/Linux远程只读诊断Agent
//
// 这是一个自主的MySQL和Linux系统性能诊断工具,使用LLM自动决策
// 需要收集哪些信息,最终生成综合诊断报告。
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"mysql-diagnostic-agent/internal/agent"
	"mysql-diagnostic-agent/internal/config"
	"mysql-diagnostic-agent/internal/mysql"
	"mysql-diagnostic-agent/internal/report"
	"mysql-diagnostic-agent/internal/ssh"
)

func main() {
	// 设置日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 创建上下文,支持信号中断
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("收到中断信号,正在停止...")
		cancel()
	}()

	// 运行主程序
	if err := run(ctx, logger); err != nil {
		logger.Error("诊断失败", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	logger.Info("============================================================")
	logger.Info("   MySQL/Linux 远程只读诊断 Agent")
	logger.Info("============================================================")

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// 初始化工具
	logger.Info("正在初始化工具...")

	// SSH工具
	sshTool := ssh.NewClient(
		cfg.SSHHost,
		cfg.SSHPort,
		cfg.SSHUser,
		cfg.SSHPassword,
		cfg.SSHKeyPath,
	)
	sshTool.SetLogger(logger)

	// MySQL工具
	mysqlTool := mysql.NewClient(
		cfg.MySQLHost,
		cfg.MySQLPort,
		cfg.MySQLUser,
		cfg.MySQLPassword,
		cfg.MySQLDatabase,
	)
	mysqlTool.SetLogger(logger)

	// 报告生成器
	reportWriter, err := report.NewWriter("reports")
	if err != nil {
		return err
	}
	reportWriter.SetLogger(logger)

	// 测试连接
	logger.Info("正在测试连接...")

	if err := sshTool.Connect(); err != nil {
		return err
	}
	logger.Info("SSH连接: 正常")
	defer sshTool.Disconnect()

	if err := mysqlTool.Connect(); err != nil {
		return err
	}
	logger.Info("MySQL连接: 正常")
	defer mysqlTool.Disconnect()

	// 初始化并运行Agent
	logger.Info("正在初始化诊断Agent...")

	diagnosticAgent := agent.NewAgent(
		sshTool,
		mysqlTool,
		reportWriter,
		cfg.APIKey,
		cfg.BaseURL,
		cfg.Model,
		cfg.SlowQueryThreshold,
	)
	diagnosticAgent.SetLogger(logger)

	// 运行诊断
	logger.Info("开始自主诊断...")

	finalAnswer, err := diagnosticAgent.Run(ctx)
	if err != nil {
		return err
	}

	// 生成报告
	logger.Info("正在生成报告...")

	reportPath, err := reportWriter.GenerateMarkdown(
		finalAnswer,
		map[string]any{
			"目标SSH":      cfg.SSHAddr(),
			"目标MySQL":    cfg.MySQLAddr(),
			"使用模型":      cfg.Model,
			"慢查询阈值":    cfg.SlowQueryThreshold,
		},
	)
	if err != nil {
		return err
	}

	// 输出摘要
	println()
	println(reportWriter.GenerateSummaryText(finalAnswer))
	println()
	println("完整报告已保存至:", reportPath)
	println()

	logger.Info("诊断完成!")
	return nil
}
