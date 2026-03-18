"""
SSH 工具模块 - 只读 Linux 命令执行

提供安全的 SSH 连接和只读命令执行功能，
包含命令白名单和安全检查机制。
"""
import os
import logging
from typing import Optional, Tuple
import paramiko

logger = logging.getLogger(__name__)


class SSHTool:
    """
    SSH 工具类，用于执行只读的 Linux 命令

    该类提供安全的 SSH 连接管理和命令执行功能，
    所有执行的命令都会经过白名单验证。
    """

    ALLOWED_COMMANDS = {
        # 系统信息
        "top", "uptime", "uname", "hostname", "date", "ls", "cat",
        # 资源监控
        "free", "df", "du", "vmstat", "iostat", "mpstat", "sar",
        # 进程管理
        "ps", "pstree", "pgrep", "pidof", "lsof", "fuser",
        # 网络工具
        "netstat", "ss", "ip", "ifconfig", "ping", "traceroute",
        "nslookup", "dig", "nethogs", "iftop",
        # 系统日志
        "dmesg", "journalctl",
        # 系统状态
        "lscpu", "lsmem", "lsblk",
        # MySQL 相关只读命令
        "mysqladmin", "mysqldump",
        # 文本处理
        "grep", "awk", "sed", "cut", "sort", "uniq", "wc", "head", "tail",
        # 性能监控
        "iotop", "htop", "atop",
        # 服务管理
        "systemctl",
    }

    BLOCKED_SUBSTRINGS = [
        # 危险的重定向（可能覆盖文件）
        ">", ">>",
        # 输入重定向
        "<", "<<",
        # 命令分隔符（可能执行多条命令）
        "&&", "||",
        # 命令替换
        "`", "$(", "${",
        # 危险操作
        "rm -rf", "mkfs", "dd if=",
        # Fork 炸弹
        ":(){ :|:& };:",
        # 权限修改
        "chmod 777", "chown -R",
        # 破坏性操作
        "mv /", "cp /dev/null", "> /dev/sd",
        # 代码执行
        "exec", "eval", "source", ". ",
        # 权限提升
        "sudo", "su ",
        # 网络下载
        "wget ", "curl ",
        # 包管理
        "apt-get", "yum ", "dnf ", "pip ",
        # 版本控制
        "git ",
        # 容器/虚拟化
        "docker ", "kube", "virsh",
    ]

    def __init__(
        self,
        host: str,
        port: int = 22,
        username: str = "root",
        password: Optional[str] = None,
        key_path: Optional[str] = None,
    ):
        self.host = host
        self.port = port
        self.username = username
        self.password = password
        self.key_path = key_path
        self._client: Optional[paramiko.SSHClient] = None

    def connect(self) -> None:
        """建立 SSH 连接"""
        if self._client:
            return

        self._client = paramiko.SSHClient()
        self._client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

        connect_kwargs = {
            "hostname": self.host,
            "port": self.port,
            "username": self.username,
        }

        if self.key_path and os.path.exists(self.key_path):
            connect_kwargs["key_filename"] = self.key_path
        elif self.password:
            connect_kwargs["password"] = self.password

        logger.info(f"正在连接 SSH: {self.username}@{self.host}:{self.port}")
        self._client.connect(**connect_kwargs, timeout=30)

    def disconnect(self) -> None:
        """关闭 SSH 连接"""
        if self._client:
            self._client.close()
            self._client = None
            logger.info("SSH 连接已断开")

    def _validate_command(self, command: str) -> Tuple[bool, str]:
        """验证命令是否为只读且安全"""
        cmd_lower = command.lower()

        for blocked in self.BLOCKED_SUBSTRINGS:
            if blocked in cmd_lower:
                return False, f"命令包含禁止的子串: {blocked}"

        cmd_parts = command.strip().split()
        if not cmd_parts:
            return False, "空命令"

        cmd_name = cmd_parts[0].rstrip(";")  # 去除命令末尾的分号
        if "/" in cmd_name:
            cmd_name = cmd_name.split("/")[-1]

        if cmd_name == "mysqladmin":
            allowed_mysqladmin = {
                "ping", "status", "version", "extended-status",
                "processlist", "variables", "info"
            }
            for subcmd in allowed_mysqladmin:
                if subcmd in command:
                    return True, ""
            return False, "mysqladmin 子命令不在允许列表中"

        if cmd_name == "mysqldump":
            if "--no-data" not in command and "-d" not in command:
                return False, "mysqldump 必须使用 --no-data 或 -d 参数"
            return True, ""

        if cmd_name not in self.ALLOWED_COMMANDS:
            return False, f"命令不在允许列表中: {cmd_name}"

        return True, ""

    def execute(self, command: str, timeout: int = 60) -> str:
        """执行只读命令"""
        is_valid, error_msg = self._validate_command(command)
        if not is_valid:
            raise ValueError(f"命令验证失败: {error_msg}")

        if not self._client:
            self.connect()

        logger.info(f"执行 SSH 命令: {command}")

        try:
            stdin, stdout, stderr = self._client.exec_command(command, timeout=timeout)

            output = stdout.read().decode("utf-8", errors="replace")
            error = stderr.read().decode("utf-8", errors="replace")

            exit_code = stdout.channel.recv_exit_status()

            if exit_code != 0:
                logger.warning(f"命令退出码 {exit_code}: {error}")

            result = output
            if error and not output:
                result = error
            elif error:
                result = output + "\n\nSTDERR:\n" + error

            return result.strip()

        except Exception as e:
            logger.error(f"命令执行失败: {e}")
            raise RuntimeError(f"执行命令失败: {e}")

    def __enter__(self):
        self.connect()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.disconnect()
