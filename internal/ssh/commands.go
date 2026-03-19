package ssh

// 允许执行的命令白名单
var allowedCommands = map[string]struct{}{
	// 系统信息
	"top": {}, "uptime": {}, "uname": {}, "hostname": {}, "date": {}, "ls": {}, "cat": {},
	// 资源监控
	"free": {}, "df": {}, "du": {}, "vmstat": {}, "iostat": {}, "mpstat": {}, "sar": {},
	// 进程管理
	"ps": {}, "pstree": {}, "pgrep": {}, "pidof": {}, "lsof": {}, "fuser": {},
	// 网络工具
	"netstat": {}, "ss": {}, "ip": {}, "ifconfig": {}, "ping": {}, "traceroute": {},
	"nslookup": {}, "dig": {}, "nethogs": {}, "iftop": {},
	// 系统日志
	"dmesg": {}, "journalctl": {},
	// 系统状态
	"lscpu": {}, "lsmem": {}, "lsblk": {},
	// MySQL 相关只读命令
	"mysqladmin": {}, "mysqldump": {},
	// 文本处理
	"grep": {}, "awk": {}, "sed": {}, "cut": {}, "sort": {}, "uniq": {}, "wc": {}, "head": {}, "tail": {},
	// 性能监控
	"iotop": {}, "htop": {}, "atop": {},
	// 服务管理
	"systemctl": {},
}

// 禁止的子串列表
var blockedSubstrings = []string{
	// 危险的重定向（可能覆盖文件）
	">", ">>",
	// 输入重定向
	"<", "<<",
	// 命令分隔符（可能执行多条命令）
	"&&", "||",
	// 命令替换
	"`", "$(", "${",
	// 危险操作
	"rm -rf", "mkfs", "dd if=",
	// Fork 炸弹
	":(){ :|:& };:",
	// 权限修改
	"chmod 777", "chown -R",
	// 破坏性操作
	"mv /", "cp /dev/null", "> /dev/sd",
	// 代码执行
	"exec", "eval", "source", ". ",
	// 权限提升
	"sudo", "su ",
	// 网络下载
	"wget ", "curl ",
	// 包管理
	"apt-get", "yum ", "dnf ", "pip ",
	// 版本控制
	"git ",
	// 容器/虚拟化
	"docker ", "kube", "virsh",
}
