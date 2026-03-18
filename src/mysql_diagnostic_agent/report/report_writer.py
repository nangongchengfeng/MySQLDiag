"""
报告生成器模块 - 生成诊断报告

负责收集诊断过程中的观察结果，并生成
详细的 Markdown 格式诊断报告。
"""
import os
import logging
from datetime import datetime
from typing import Dict, Any, List, Optional

logger = logging.getLogger(__name__)


class ReportWriter:
    """
    诊断报告生成器

    用于记录诊断过程中的每一轮观察，并最终生成
    完整的 Markdown 格式报告。
    """

    def __init__(self, output_dir: str = "reports"):
        self.output_dir = output_dir
        self.observations: List[Dict[str, Any]] = []
        self.start_time = datetime.now()
        self._ensure_output_dir()

    def _ensure_output_dir(self) -> None:
        """确保输出目录存在"""
        if not os.path.exists(self.output_dir):
            os.makedirs(self.output_dir)
            logger.info(f"已创建输出目录: {self.output_dir}")

    def add_observation(
        self,
        tool: str,
        action: str,
        input_data: str,
        observation: str,
        round_num: int,
    ) -> None:
        """添加一条观察记录到报告"""
        self.observations.append({
            "round": round_num,
            "tool": tool,
            "action": action,
            "input": input_data,
            "observation": observation,
            "timestamp": datetime.now().isoformat(),
        })
        logger.info(f"[第 {round_num} 轮] {tool}.{action} - 观察已记录")

    def _generate_filename(self) -> str:
        """生成唯一的报告文件名"""
        timestamp = self.start_time.strftime("%Y%m%d_%H%M%S")
        return f"diagnostic_report_{timestamp}.md"

    def _format_code_block(self, content: str, language: str = "") -> str:
        """将内容格式化为 Markdown 代码块"""
        if not content:
            return ""
        if len(content) > 5000:
            content = content[:5000] + "\n... [已截断]"
        return f"```{language}\n{content}\n```"

    def generate_markdown(
        self,
        final_answer: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> str:
        """生成 Markdown 格式的完整报告"""
        end_time = datetime.now()
        duration = (end_time - self.start_time).total_seconds()

        filename = self._generate_filename()
        filepath = os.path.join(self.output_dir, filename)

        lines = []
        lines.append("# MySQL/Linux 诊断报告")
        lines.append("")
        lines.append(f"- **生成时间**: {self.start_time.strftime('%Y-%m-%d %H:%M:%S')}")
        lines.append(f"- **总耗时**: {duration:.2f} 秒")
        lines.append(f"- **诊断轮次**: {max((o['round'] for o in self.observations), default=0)}")
        lines.append("")

        if metadata:
            lines.append("## 配置信息")
            lines.append("")
            for key, value in metadata.items():
                lines.append(f"- **{key}**: {value}")
            lines.append("")

        lines.append("## 最终诊断")
        lines.append("")
        lines.append(final_answer)
        lines.append("")

        lines.append("## 诊断过程")
        lines.append("")

        if not self.observations:
            lines.append("*没有记录观察数据。*")
            lines.append("")
        else:
            rounds: Dict[int, List[Dict[str, Any]]] = {}
            for obs in self.observations:
                r = obs["round"]
                if r not in rounds:
                    rounds[r] = []
                rounds[r].append(obs)

            for round_num in sorted(rounds.keys()):
                lines.append(f"### 第 {round_num} 轮")
                lines.append("")

                for obs in rounds[round_num]:
                    lines.append(f"**工具**: `{obs['tool']}.{obs['action']}`")
                    lines.append("")
                    lines.append("**输入**:")
                    lines.append("")
                    lines.append(self._format_code_block(str(obs["input"])))
                    lines.append("")
                    lines.append("**观察结果**:")
                    lines.append("")
                    lines.append(self._format_code_block(str(obs["observation"])))
                    lines.append("")

        lines.append("---")
        lines.append("")
        lines.append("*由 MySQL 诊断 Agent 自动生成*")

        content = "\n".join(lines)
        with open(filepath, "w", encoding="utf-8") as f:
            f.write(content)

        logger.info(f"报告已写入: {filepath}")
        return filepath

    def generate_summary_text(self, final_answer: str) -> str:
        """生成简洁的文本摘要"""
        lines = []
        lines.append("=" * 60)
        lines.append("        MySQL/Linux 诊断报告摘要")
        lines.append("=" * 60)
        lines.append("")
        lines.append("最终诊断:")
        lines.append("-" * 40)
        lines.append(final_answer)
        lines.append("")
        lines.append("=" * 60)
        lines.append(f"总诊断轮次: {max((o['round'] for o in self.observations), default=0)}")
        return "\n".join(lines)
