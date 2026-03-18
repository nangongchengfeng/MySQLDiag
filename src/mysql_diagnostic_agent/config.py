"""
配置管理模块

提供统一的配置加载和验证功能。
"""
import os
import logging
from typing import Dict, Optional
from dataclasses import dataclass
from dotenv import load_dotenv

logger = logging.getLogger(__name__)


@dataclass
class Config:
    """配置数据类"""
    # 必填配置（无默认值）
    api_key: str
    ssh_host: str
    mysql_host: str

    # API 配置（有默认值）
    base_url: str = "https://api.siliconflow.cn/v1/"
    model: str = "deepseek-ai/DeepSeek-V3.2"

    # SSH 配置（有默认值）
    ssh_port: int = 22
    ssh_user: str = "root"
    ssh_password: Optional[str] = None
    ssh_key_path: Optional[str] = None

    # MySQL 配置（有默认值）
    mysql_port: int = 3306
    mysql_user: str = "root"
    mysql_password: str = ""
    mysql_database: Optional[str] = None

    # 诊断配置（有默认值）
    slow_query_threshold: float = 0.5


def load_config() -> Config:
    """
    从环境变量加载配置

    Returns:
        Config 对象
    """
    load_dotenv()

    return Config(
        # API 配置
        api_key=_get_env_required("SILICONFLOW_API_KEY"),
        base_url=os.getenv("SILICONFLOW_BASE_URL", "https://api.siliconflow.cn/v1/"),
        model=os.getenv("SILICONFLOW_MODEL", "deepseek-ai/DeepSeek-V3.2"),

        # SSH 配置
        ssh_host=_get_env_required("SSH_HOST"),
        ssh_port=int(os.getenv("SSH_PORT", "22")),
        ssh_user=os.getenv("SSH_USER", "root"),
        ssh_password=os.getenv("SSH_PASSWORD"),
        ssh_key_path=os.getenv("SSH_KEY_PATH"),

        # MySQL 配置
        mysql_host=_get_env_required("MYSQL_HOST"),
        mysql_port=int(os.getenv("MYSQL_PORT", "3306")),
        mysql_user=os.getenv("MYSQL_USER", "root"),
        mysql_password=os.getenv("MYSQL_PASSWORD", ""),
        mysql_database=os.getenv("MYSQL_DATABASE"),

        # 诊断配置
        slow_query_threshold=float(os.getenv("SLOW_QUERY_THRESHOLD", "0.5")),
    )


def _get_env_required(name: str) -> str:
    """获取必需的环境变量"""
    value = os.getenv(name)
    if not value:
        raise ValueError(f"缺少必需的环境变量: {name}")
    return value
