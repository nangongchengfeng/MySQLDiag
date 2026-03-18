"""
工具调用模块

定义工具调用数据类。
"""
from dataclasses import dataclass
from typing import Dict


@dataclass
class ToolCall:
    """工具调用决策"""
    tool: str
    action: str
    input_data: str

    def to_dict(self) -> Dict[str, str]:
        """转换为字典格式"""
        return {
            "tool": self.tool,
            "action": self.action,
            "input": self.input_data,
        }
