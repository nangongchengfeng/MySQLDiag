"""
MySQL/Linux 远程只读诊断 Agent

一个自主的 MySQL 和 Linux 系统性能诊断工具，使用 LLM 自动决策
需要收集哪些信息，最终生成综合诊断报告。
"""

__version__ = "1.0.0"
__author__ = "MySQLDiag Team"

from .tools import SSHTool, MySQLTool
from .report import ReportWriter
from .agent import DiagnosticAgent
from .config import Config, load_config

__all__ = [
    "SSHTool",
    "MySQLTool",
    "ReportWriter",
    "DiagnosticAgent",
    "Config",
    "load_config",
]
