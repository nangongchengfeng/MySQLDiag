"""
诊断 Agent - 核心编排逻辑

使用 LLM 自主决定需要收集哪些信息，
何时可以得出诊断结论。提供中文界面和提示词。
"""
import os
import logging
import json
from typing import Dict, Any, List, Optional
from dotenv import load_dotenv
import httpx

from ..tools import SSHTool, MySQLTool
from ..report import ReportWriter
from ..core import ToolCall

logger = logging.getLogger(__name__)
load_dotenv()


class DiagnosticAgent:
    """
    自主 MySQL/Linux 诊断 Agent

    使用 LLM 来决定收集什么信息，何时可以结束诊断。
    会自动检查 MySQL 和系统两方面的指标。
    """

    MAX_ROUNDS = 30

    SYSTEM_PROMPT = """你是一位专业的 MySQL 数据库和 Linux 系统性能诊断专家。
你的任务是通过收集和分析信息，诊断数据库和系统的性能问题。

【可用工具】

你有以下三个工具可以使用：

1. mysql_query - 执行只读 MySQL 查询
   - 输入: SQL 查询语句（仅允许 SELECT, SHOW, DESCRIBE, EXPLAIN）
   - 用途: 检查 MySQL 状态、变量、进程列表、慢查询、锁信息等
   - 提示: 可以一次执行多条用分号分隔的查询
   - 常用查询示例:
     * SHOW GLOBAL STATUS LIKE 'Slow_queries'
     * SHOW VARIABLES LIKE 'long_query_time'
     * SHOW GLOBAL STATUS LIKE 'Questions'
     * SHOW GLOBAL STATUS LIKE 'Uptime'
     * SHOW FULL PROCESSLIST
     * SHOW ENGINE INNODB STATUS
     * SHOW GLOBAL STATUS LIKE 'Threads_%'
     * SHOW GLOBAL STATUS LIKE 'Connections'
     * SHOW GLOBAL STATUS LIKE 'Aborted_%'
     * SHOW GLOBAL VARIABLES
     * SHOW OPEN TABLES WHERE In_use > 0

2. ssh_exec - 通过 SSH 执行只读 Linux 命令
   - 输入: Linux 命令（仅允许只读命令）
   - 用途: 检查系统资源：CPU、内存、磁盘、网络、进程等
   - 常用命令示例:
     * top -bn1 | head -50
     * free -h
     * df -h
     * uptime
     * vmstat 1 3
     * iostat -x 1 3
     * netstat -tuln
     * netstat -an | grep ESTABLISHED | wc -l
     * ps aux --sort=-%cpu | head -20
     * ps aux --sort=-%mem | head -20
     * dmesg | tail -50
     * ss -s

3. final_answer - 结束诊断并给出结论
   - 输入: 综合诊断报告和优化建议
   - 使用时机: 当你收集了足够的信息，可以给出完整诊断时

【诊断流程建议】

请按以下思路进行诊断，但可以根据实际情况调整：

第1阶段 - 基础检查（建议第1-3轮）：
1. 慢查询统计 - Slow_queries, long_query_time
2. MySQL 运行时间 - Uptime
3. 查询量统计 - Questions, 计算 QPS
4. 连接数统计 - Threads_connected, Max_used_connections
5. 系统基本信息 - uptime, free, df

第2阶段 - 深入检查（建议第4-10轮）：
1. 进程列表 - SHOW FULL PROCESSLIST
2. InnoDB 状态 - SHOW ENGINE INNODB STATUS
3. 锁等待信息
4. CPU 使用率 - top, vmstat
5. 内存使用 - free, ps
6. 磁盘 I/O - iostat, df
7. 网络连接 - netstat, ss

第3阶段 - 针对性分析（按需进行）：
1. 表缓存状态
2. 查询缓存（如果启用）
3. 临时表使用情况
4. 排序统计
5. 特定慢查询的 EXPLAIN

【重要规则】

1. 只能使用上面列出的三个工具
2. 所有 MySQL 查询必须是只读的（SELECT, SHOW, DESCRIBE, EXPLAIN）
3. 所有 SSH 命令必须是只读的（不能修改系统）
4. 每个命令只能执行一个操作，禁止使用分号 `;`、`&&`、`||` 等命令分隔符
5. 不要在 MySQL 查询中使用 `\\G` 语法（这是命令行语法，不支持）
6. 从基础检查开始，然后根据发现深入挖掘
7. 必须全面检查：既要检查 MySQL，也要检查系统资源
8. 当你认为已经收集了足够信息时，使用 final_answer

【响应格式】

你的回答必须是纯 JSON 格式，不要包含其他文字：

{
  "tool": "mysql_query|ssh_exec|final_answer",
  "action": "具体动作描述",
  "input": "要执行的查询或命令"
}

【最终诊断报告要求】

当使用 final_answer 时，请提供包含以下内容的综合报告：

1. 总体评价 - 健康状态评分（0-100分）
2. 慢查询分析 - 数量、阈值、占比
3. QPS 和负载分析
4. 连接和线程状态
5. 锁等待分析
6. 内存使用状况
7. CPU 使用状况
8. 磁盘空间和 I/O
9. 主要问题列表（按严重程度排序）
10. 具体的优化建议

现在，请开始诊断工作！"""

    def __init__(
        self,
        ssh_tool: SSHTool,
        mysql_tool: MySQLTool,
        report_writer: ReportWriter,
        api_key: str,
        base_url: str,
        model: str,
        slow_query_threshold: float = 0.5,
    ):
        self.ssh_tool = ssh_tool
        self.mysql_tool = mysql_tool
        self.report_writer = report_writer
        self.api_key = api_key
        self.base_url = base_url
        self.model = model
        self.slow_query_threshold = slow_query_threshold
        self.messages: List[Dict[str, str]] = []
        self.current_round = 0

    def _build_initial_context(self) -> str:
        """构建初始上下文"""
        return f"""MySQL/Linux 诊断会话开始

【配置信息】
- 慢查询阈值: {self.slow_query_threshold} 秒
- 目标: MySQL 和 Linux 系统健康检查

请开始诊断过程。建议按以下顺序进行：
1. 检查慢查询统计信息
2. 检查 MySQL 状态和配置变量
3. 检查系统资源使用情况

请确保收集足够全面的信息后再给出最终诊断。
"""

    def _call_llm(self, user_message: str) -> ToolCall:
        """调用 LLM 获取下一步工具决策"""
        self.messages.append({"role": "user", "content": user_message})

        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }

        payload = {
            "model": self.model,
            "messages": [
                {"role": "system", "content": self.SYSTEM_PROMPT}
            ] + self.messages,
            "temperature": 0.3,
            "max_tokens": 3000,
        }

        url = f"{self.base_url.rstrip('/')}/chat/completions"

        logger.debug(f"调用 LLM API: {url}")

        try:
            with httpx.Client(timeout=120) as client:
                response = client.post(url, headers=headers, json=payload)
                response.raise_for_status()
                result = response.json()

            assistant_message = result["choices"][0]["message"]["content"]
            self.messages.append({"role": "assistant", "content": assistant_message})

            assistant_message = assistant_message.strip()
            if assistant_message.startswith("```json"):
                assistant_message = assistant_message[7:]
            if assistant_message.startswith("```"):
                assistant_message = assistant_message[3:]
            if assistant_message.endswith("```"):
                assistant_message = assistant_message[:-3]
            assistant_message = assistant_message.strip()

            parsed = json.loads(assistant_message)
            return ToolCall(
                tool=parsed["tool"],
                action=parsed.get("action", parsed["tool"]),
                input_data=parsed["input"],
            )

        except Exception as e:
            logger.error(f"LLM 调用失败: {e}", exc_info=True)
            return ToolCall(
                tool="final_answer",
                action="结束诊断",
                input_data=f"诊断因错误中断: {e}\n\n请查看报告中的观察数据进行手动分析。",
            )

    def _execute_tool(self, tool_call: ToolCall) -> str:
        """执行工具调用并返回结果"""
        try:
            if tool_call.tool == "mysql_query":
                result = self.mysql_tool.query(tool_call.input_data)
                if result:
                    lines = []
                    headers = list(result[0].keys())
                    lines.append(" | ".join(str(h) for h in headers))
                    lines.append("-+-".join("-" * len(str(h)) for h in headers))
                    for row in result:
                        lines.append(" | ".join(str(row.get(h, "")) for h in headers))
                    return "\n".join(lines)
                return "查询执行成功，无结果返回。"

            elif tool_call.tool == "ssh_exec":
                return self.ssh_tool.execute(tool_call.input_data)

            elif tool_call.tool == "final_answer":
                return "收到最终诊断结果。"

            else:
                return f"未知工具: {tool_call.tool}"

        except Exception as e:
            logger.error(f"工具执行失败: {e}", exc_info=True)
            return f"错误: {str(e)}"

    def run(self) -> str:
        """运行完整的诊断流程"""
        logger.info("诊断 Agent 启动...")
        self.current_round = 0

        user_message = self._build_initial_context()

        while self.current_round < self.MAX_ROUNDS:
            self.current_round += 1
            logger.info(f"--- 第 {self.current_round} 轮诊断 ---")

            tool_call = self._call_llm(user_message)
            logger.info(f"决策: {tool_call.tool} - {tool_call.action}")

            if tool_call.tool == "final_answer":
                final_answer = tool_call.input_data
                self.report_writer.add_observation(
                    tool="agent",
                    action="final_answer",
                    input_data="",
                    observation=final_answer,
                    round_num=self.current_round,
                )
                logger.info("已得出最终诊断。")
                return final_answer

            observation = self._execute_tool(tool_call)

            self.report_writer.add_observation(
                tool=tool_call.tool,
                action=tool_call.action,
                input_data=tool_call.input_data,
                observation=observation,
                round_num=self.current_round,
            )

            user_message = f"""来自 {tool_call.tool}.{tool_call.action} 的观察结果:

【输入】
{tool_call.input_data}

【结果】
{observation}

接下来你想做什么？如果已经收集了足够的信息，请使用 final_answer 给出诊断结论。
"""

        final_answer = "诊断已达到最大轮次限制仍未结束。请查看已收集的观察数据进行手动分析。"
        logger.warning("已达到最大诊断轮次。")
        return final_answer
