"""
MySQL 工具模块 - 只读 MySQL 查询

提供安全的 MySQL 连接和只读查询功能，
包含 SQL 白名单和安全检查机制，同时支持批量查询。
"""
import logging
from typing import Optional, List, Tuple, Any
import pymysql
from pymysql.cursors import DictCursor

logger = logging.getLogger(__name__)


class MySQLTool:
    """
    MySQL 工具类，用于执行只读查询

    该类提供安全的 MySQL 连接管理和查询执行功能，
    所有执行的 SQL 都会经过白名单验证。
    支持一次执行多条用分号分隔的查询。
    """

    ALLOWED_STATEMENTS = {
        "SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "USE",
        "HELP", "CHECKSUM", "CHECK", "ANALYZE"
    }

    BLOCKED_KEYWORDS = [
        "INSERT", "UPDATE", "DELETE", "REPLACE",
        "DROP", "TRUNCATE", "ALTER", "CREATE", "RENAME",
        "GRANT", "REVOKE",
        "SET",
        "LOAD",
        "LOCK", "UNLOCK",
        "START", "COMMIT", "ROLLBACK", "SAVEPOINT", "RELEASE",
        "PREPARE", "EXECUTE", "DEALLOCATE",
        "OPTIMIZE", "REPAIR", "USE FRM",
        "BACKUP", "RESTORE", "IMPORT", "EXPORT",
        "FLUSH", "RESET", "SHUTDOWN", "KILL", "PURGE", "CHANGE",
        "INSTALL", "UNINSTALL", "PLUGIN"
    ]

    def __init__(
        self,
        host: str,
        port: int = 3306,
        username: str = "root",
        password: str = "",
        database: Optional[str] = None,
    ):
        self.host = host
        self.port = port
        self.username = username
        self.password = password
        self.database = database
        self._connection: Optional[pymysql.Connection] = None

    def connect(self) -> None:
        """建立 MySQL 连接"""
        if self._connection:
            return

        logger.info(f"正在连接 MySQL: {self.username}@{self.host}:{self.port}")

        self._connection = pymysql.connect(
            host=self.host,
            port=self.port,
            user=self.username,
            password=self.password,
            database=self.database,
            cursorclass=DictCursor,
            charset="utf8mb4",
            connect_timeout=30,
            read_timeout=60,
            write_timeout=60,
        )

    def disconnect(self) -> None:
        """关闭 MySQL 连接"""
        if self._connection:
            self._connection.close()
            self._connection = None
            logger.info("MySQL 连接已断开")

    def _validate_query(self, query: str) -> Tuple[bool, str]:
        """验证单条查询是否为只读且安全"""
        query_upper = query.strip().upper()

        if not query_upper:
            return False, "空查询"

        lines = []
        for line in query_upper.split("\n"):
            line = line.split("--", 1)[0].split("#", 1)[0].strip()
            if line:
                lines.append(line)
        clean_query = " ".join(lines)

        for keyword in self.BLOCKED_KEYWORDS:
            if f" {keyword} " in f" {clean_query} ":
                return False, f"查询包含禁止的关键字: {keyword}"
            if clean_query.startswith(f"{keyword} "):
                return False, f"查询以禁止关键字开头: {keyword}"

        first_word = clean_query.split()[0] if clean_query else ""

        if first_word == "ANALYZE" and "TABLE" in clean_query:
            return True, ""

        if first_word not in self.ALLOWED_STATEMENTS:
            return False, f"语句类型不在允许列表中: {first_word}"

        return True, ""

    def _split_multi_query(self, sql: str) -> List[str]:
        """将多条用分号分隔的 SQL 拆分为单条语句"""
        queries = []
        current_query = []
        in_string = False
        string_char = None
        escape_next = False

        for char in sql:
            if escape_next:
                current_query.append(char)
                escape_next = False
                continue

            if char == "\\":
                current_query.append(char)
                escape_next = True
                continue

            if char in ("'", '"'):
                if not in_string:
                    in_string = True
                    string_char = char
                elif char == string_char:
                    in_string = False
                    string_char = None
                current_query.append(char)
                continue

            if char == ";" and not in_string:
                query = "".join(current_query).strip()
                if query:
                    queries.append(query)
                current_query = []
                continue

            current_query.append(char)

        query = "".join(current_query).strip()
        if query:
            queries.append(query)

        return queries

    def query(self, sql: str, params: Optional[Tuple] = None) -> List[dict]:
        """执行只读 MySQL 查询"""
        queries = self._split_multi_query(sql)

        if not queries:
            return []

        # 多条查询时不支持参数
        if len(queries) > 1:
            if params:
                logger.warning("多条查询不支持参数，参数将被忽略")
            all_results = []
            for i, single_query in enumerate(queries, 1):
                logger.info(f"执行多查询 [{i}/{len(queries)}]: {single_query}")
                results = self._execute_single_query(single_query)
                if results:
                    all_results.extend(results)
            return all_results

        return self._execute_single_query(queries[0], params)

    def _execute_single_query(self, sql: str, params: Optional[Tuple] = None) -> List[dict]:
        """执行单条查询（内部方法）"""
        # 去除 MySQL 命令行语法 \G（不支持这种语法）
        sql = sql.rstrip().rstrip("\\G").strip()
        is_valid, error_msg = self._validate_query(sql)
        if not is_valid:
            raise ValueError(f"查询验证失败: {error_msg}")

        if not self._connection:
            self.connect()

        logger.info(f"执行 MySQL 查询: {sql}")

        try:
            with self._connection.cursor() as cursor:
                # 只有在提供参数时才使用参数化查询
                if params:
                    cursor.execute(sql, params)
                else:
                    cursor.execute(sql)
                result = cursor.fetchall()
                return list(result) if result else []
        except Exception as e:
            logger.error(f"查询执行失败: {e}")
            raise RuntimeError(f"执行查询失败: {e}")

    def query_one(self, sql: str, params: Optional[Tuple] = None) -> Optional[dict]:
        """执行查询并返回单行结果"""
        results = self.query(sql, params)
        return results[0] if results else None

    def query_value(self, sql: str, params: Optional[Tuple] = None) -> Any:
        """执行查询并返回第一列的单个值"""
        row = self.query_one(sql, params)
        if row:
            return next(iter(row.values()))
        return None

    def show_variables(self, like: Optional[str] = None) -> dict:
        """获取 MySQL 系统变量"""
        if like:
            sql = "SHOW VARIABLES LIKE %s"
            rows = self.query(sql, (like,))
        else:
            rows = self.query("SHOW VARIABLES")
        return {row["Variable_name"]: row["Value"] for row in rows}

    def show_status(self, like: Optional[str] = None) -> dict:
        """获取 MySQL 状态信息"""
        if like:
            sql = "SHOW GLOBAL STATUS LIKE %s"
            rows = self.query(sql, (like,))
        else:
            rows = self.query("SHOW GLOBAL STATUS")
        return {row["Variable_name"]: row["Value"] for row in rows}

    def show_processlist(self) -> List[dict]:
        """获取当前进程列表"""
        return self.query("SHOW FULL PROCESSLIST")

    def show_engine_status(self) -> Optional[dict]:
        """获取 InnoDB 引擎状态"""
        rows = self.query("SHOW ENGINE INNODB STATUS")
        if rows:
            return rows[0]
        return None

    def __enter__(self):
        self.connect()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.disconnect()
