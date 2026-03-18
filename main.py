#!/usr/bin/env python3
"""
MySQL/Linux 远程只读诊断 Agent

这是一个自主的 MySQL 和 Linux 系统性能诊断工具，使用 LLM 自动决策
需要收集哪些信息，最终生成综合诊断报告。
"""
import sys
import logging

from src.mysql_diagnostic_agent import (
    SSHTool,
    MySQLTool,
    ReportWriter,
    DiagnosticAgent,
)
from src.mysql_diagnostic_agent.config import load_config


def setup_logging() -> None:
    """配置日志系统"""
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
        handlers=[logging.StreamHandler(sys.stdout)],
    )


def main() -> int:
    """主入口函数"""
    setup_logging()
    logger = logging.getLogger(__name__)

    try:
        logger.info("=" * 60)
        logger.info("   MySQL/Linux 远程只读诊断 Agent")
        logger.info("=" * 60)

        # 加载配置
        config = load_config()

        # 初始化工具
        logger.info("正在初始化工具...")

        ssh_tool = SSHTool(
            host=config.ssh_host,
            port=config.ssh_port,
            username=config.ssh_user,
            password=config.ssh_password,
            key_path=config.ssh_key_path,
        )

        mysql_tool = MySQLTool(
            host=config.mysql_host,
            port=config.mysql_port,
            username=config.mysql_user,
            password=config.mysql_password,
            database=config.mysql_database,
        )

        report_writer = ReportWriter(output_dir="reports")

        # 测试连接
        logger.info("正在测试连接...")
        ssh_tool.connect()
        logger.info("SSH 连接: 正常")

        mysql_tool.connect()
        logger.info("MySQL 连接: 正常")

        # 初始化并运行 Agent
        logger.info("正在初始化诊断 Agent...")

        agent = DiagnosticAgent(
            ssh_tool=ssh_tool,
            mysql_tool=mysql_tool,
            report_writer=report_writer,
            api_key=config.api_key,
            base_url=config.base_url,
            model=config.model,
            slow_query_threshold=config.slow_query_threshold,
        )

        # 运行诊断
        logger.info("开始自主诊断...")
        final_answer = agent.run()

        # 生成报告
        logger.info("正在生成报告...")
        report_path = report_writer.generate_markdown(
            final_answer=final_answer,
            metadata={
                "目标 SSH": f"{config.ssh_host}:{config.ssh_port}",
                "目标 MySQL": f"{config.mysql_host}:{config.mysql_port}",
                "使用模型": config.model,
                "慢查询阈值": f"{config.slow_query_threshold}秒",
            },
        )

        # 输出摘要
        print()
        print(report_writer.generate_summary_text(final_answer))
        print()
        print(f"完整报告已保存至: {report_path}")
        print()

        # 清理
        ssh_tool.disconnect()
        mysql_tool.disconnect()

        logger.info("诊断完成！")
        return 0

    except KeyboardInterrupt:
        logger.info("诊断被用户中断。")
        return 130
    except Exception as e:
        logger.error(f"诊断失败: {e}", exc_info=True)
        return 1


if __name__ == "__main__":
    sys.exit(main())
