# 从零开发AI诊断Agent：拆解LLM+Tools+Prompt三大核心

## 引言

不知道你有没有这种感觉——现在用 Claude Code 这类 AI 工具时，总觉得它像变魔术一样。你丢一个问题进去，它就能自动定位代码 bug、修改文件、甚至操作服务器。前阵子我做了一个 MySQL 诊断 Agent，用户只需要敲一行命令，它就能自己查慢查询、看系统负载、分析锁等待，最后给你一份完整的诊断报告。

今天这篇文章，我就把这个项目拆解开，跟你聊聊 Agent 开发的那些事儿。相信我，理解了背后的原理，你会发现 AI 工具其实没那么神秘。

### 我们面临的痛点

作为开发者或 DBA，你是不是也遇到过这些情况：

- 数据库慢了，要查一堆命令：先看慢查询，再看 QPS，然后查进程列表，还要看系统负载...
- 每次排查都要重复同样的操作，记不住那么多命令和指标
- 新手想排查问题，根本不知道从何下手
- 想做自动化诊断，但硬编码的规则又太死板

这就是我开发这个 Agent 的初衷——让 AI 来帮我们完成这些繁琐的排查工作。

### 你能从这篇文章学到什么

读完这篇文章，你会：
- 理解 Agent 的三大核心组件：LLM、Tools、Prompt
- 知道为什么 80% 的工作其实是在写 Tools
- 学会如何用 Python 开发一个实用的 Agent
- 掌握安全边界的设计（只读很重要！）

## 一、Agent 是什么？先从一个真实的诊断流程看起

在讲技术细节之前，我先给你展示一下这个 Agent 实际运行时是什么样的。

### 一个真实的诊断会话

用户只需要输入：
```bash
python main.py
```

然后 Agent 就开始自动工作了：

**第 1 轮**：LLM 决定先查慢查询
```
Action: mysql_query
Input: SHOW GLOBAL STATUS LIKE 'Slow_queries'
Observation: Slow_queries=21
```

**第 2 轮**：觉得还需要知道慢查询阈值
```
Action: mysql_query
Input: SHOW VARIABLES LIKE 'long_query_time'
Observation: long_query_time=0.5
```

**第 3-8 轮**：继续查 QPS、线程状态、锁信息、查询缓存...

**第 9-12 轮**：LLM 觉得还得看看系统资源，自动通过 SSH 执行 top、free 等命令

**第 18 轮**：LLM 终于认为信息足够了，输出最终诊断报告

整个过程中，你只需要敲一行命令，Agent 就自动完成了 18 轮诊断。这就是 Agent 的魅力所在。

### 核心思路：把决策权交给 LLM

传统的自动化脚本是这样的：
```python
def diagnose():
    check_slow_queries()
    check_qps()
    check_system_load()
    # ... 硬编码一系列检查
    return report()
```

而 Agent 的思路是这样的：
```python
def diagnose():
    while not enough_info():
        decision = llm_decide_what_to_do_next()  # 让 LLM 决定下一步
        result = execute_tool(decision)            # 执行工具
        add_to_context(result)                      # 把结果加入上下文
    return llm_generate_report()                    # 让 LLM 生成报告
```

看到区别了吗？前者是你告诉程序每一步做什么，后者是你告诉程序目标是什么，让它自己决定怎么做。

## 二、Agent 的三大核心组件

其实开发 Agent 说复杂也复杂，说简单也简单。拆开来看，核心就是三个部分：

1. **LLM** - 大脑，负责做决策
2. **Tools** - 手脚，负责执行具体操作
3. **Prompt** - 灵魂，告诉 LLM 它是谁、能做什么

接下来我逐个跟你聊。

### 2.1 LLM：Agent 的大脑

LLM 是整个 Agent 的决策中心。在这个项目里，我用的是 SiliconFlow 的 DeepSeek-V3.2 API。

为什么用 API 而不是直接用 Claude Code 界面？因为通过 API，我们可以把 LLM 集成到我们的程序里，让它按我们的逻辑来工作。

调用 LLM 的代码其实很简单：

```python
def _call_llm(self, user_message: str) -> ToolCall:
    # 构建请求消息
    self.messages.append({"role": "user", "content": user_message})

    # 调用 API
    payload = {
        "model": self.model,
        "messages": [{"role": "system", "content": self.SYSTEM_PROMPT}] + self.messages,
        "temperature": 0.3,  # 温度低一点，输出更稳定
        "max_tokens": 3000,
    }

    with httpx.Client(timeout=120) as client:
        response = client.post(url, headers=headers, json=payload)
        result = response.json()

    # 解析返回的工具调用决策
    return parse_tool_call(result)
```

就这么简单——把上下文发给 LLM，它返回一个决策，告诉我们下一步该用什么工具、做什么操作。

但这里有个关键点：LLM 本身不会直接操作数据库或服务器，它只会"说"要做什么。真正执行的是我们接下来要讲的 Tools。

### 2.2 Tools：Agent 的手脚（这才是 80% 的工作）

很多人以为开发 Agent 的重点是调 LLM、写 Prompt。其实我的体会正好相反——**80% 的工作都在写 Tools**。

为什么？因为 LLM 虽然聪明，但它"说"的东西不一定能直接执行。你需要把它的想法转换成真正安全、可靠的代码。

在这个项目里，我写了三个核心工具：

#### 工具一：SSHTool - 安全的 Linux 命令执行

第一个工具是 SSH 执行器。但这里有个大问题——你敢让 LLM 随便执行 Linux 命令吗？要是它说个 `rm -rf /` 怎么办？

所以安全是第一位的。我的做法是**白名单机制**：

```python
class SSHTool:
    # 只允许这些只读命令
    ALLOWED_COMMANDS = {
        "top", "free", "df", "du", "ps", "netstat", "ss", "uptime",
        "vmstat", "iostat", "mpstat", "sar", "dmesg", "ls", "cat",
        # ... 更多只读命令
    }

    # 绝对禁止这些危险子串
    BLOCKED_SUBSTRINGS = [
        ">", ">>", "<", "<<", "|", ";", "&&", "||",  # 重定向和管道
        "rm -rf", "mkfs", "dd if=",                   # 危险操作
        "exec", "eval", "sudo", "su ",                 # 权限提升
        # ... 更多
    ]

    def _validate_command(self, command: str) -> Tuple[bool, str]:
        # 先检查有没有危险子串
        for blocked in self.BLOCKED_SUBSTRINGS:
            if blocked in command.lower():
                return False, f"命令包含禁止的子串: {blocked}"

        # 再检查命令名是否在白名单里
        cmd_name = command.strip().split()[0]
        if "/" in cmd_name:
            cmd_name = cmd_name.split("/")[-1]

        if cmd_name not in self.ALLOWED_COMMANDS:
            return False, f"命令不在允许列表中: {cmd_name}"

        return True, ""
```

这样一来，LLM 只能执行我们允许的只读命令，完全不用担心安全问题。

#### 工具二：MySQLTool - 只读的数据库查询

第二个工具是 MySQL 查询器。同样，我们也需要严格的安全控制。

除了白名单，我还加了一个实用的功能——**支持多条 SQL 执行**。因为 LLM 有时候会想一次查多个指标，比如：

```sql
SHOW GLOBAL STATUS LIKE 'Slow_queries';
SHOW VARIABLES LIKE 'long_query_time';
SHOW GLOBAL STATUS LIKE 'Questions';
```

默认情况下，PyMySQL 不支持一次执行多条 SQL。所以我写了一个解析器：

```python
def _split_multi_query(self, sql: str) -> List[str]:
    """智能拆分多条 SQL，正确处理字符串中的分号"""
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
            # 处理字符串边界
            if not in_string:
                in_string = True
                string_char = char
            elif char == string_char:
                in_string = False
                string_char = None
            current_query.append(char)
            continue

        if char == ";" and not in_string:
            # 只有不在字符串中的分号才是语句分隔符
            query = "".join(current_query).strip()
            if query:
                queries.append(query)
            current_query = []
            continue

        current_query.append(char)

    # 处理最后一条语句
    query = "".join(current_query).strip()
    if query:
        queries.append(query)

    return queries
```

这个函数会正确处理 SQL 字符串中的分号，不会把 `SELECT 'a;b' FROM t` 拆成两条语句。

#### 工具三：ReportWriter - 报告生成器

第三个工具是报告生成器。它的工作是把每一轮的观察结果记录下来，最后生成一份漂亮的 Markdown 报告。

```python
class ReportWriter:
    def add_observation(self, tool: str, action: str,
                       input_data: str, observation: str, round_num: int):
        """记录一轮观察"""
        self.observations.append({
            "round": round_num,
            "tool": tool,
            "action": action,
            "input": input_data,
            "observation": observation,
            "timestamp": datetime.now().isoformat(),
        })

    def generate_markdown(self, final_answer: str,
                         metadata: Optional[Dict] = None) -> str:
        """生成完整的 Markdown 报告"""
        # ... 组织报告内容
        # 包含：配置信息、最终诊断、每一轮的详细观察
        # ...

        with open(filepath, "w", encoding="utf-8") as f:
            f.write(content)

        return filepath
```

好的工具设计应该是这样的：**每个工具只做一件事，但把这件事做扎实**。

### 2.3 Prompt：Agent 的灵魂

有了 LLM 和 Tools，还需要告诉 LLM 怎么用它们。这就是 Prompt 的作用。

Prompt 是 Agent 的"使用说明书"，它要回答三个问题：
1. **你是谁**？（角色定位）
2. **你能做什么**？（工具说明）
3. **你应该怎么做**？（任务指南）

在这个项目里，我的 System Prompt 是全中文的，因为目标用户是中文用户。我把它分成了几个部分：

#### 第一部分：角色定位

```python
SYSTEM_PROMPT = """你是一位专业的 MySQL 数据库和 Linux 系统性能诊断专家。
你的任务是通过收集和分析信息，诊断数据库和系统的性能问题。
"""
```

先给 LLM 一个清晰的角色——你是专家，不是助手。这样它会更专业、更自信地做决策。

#### 第二部分：工具说明

这部分要详细，因为 LLM 需要准确理解每个工具的能力：

```python
【可用工具】

你有以下三个工具可以使用：

1. mysql_query - 执行只读 MySQL 查询
   - 输入: SQL 查询语句（仅允许 SELECT, SHOW, DESCRIBE, EXPLAIN）
   - 用途: 检查 MySQL 状态、变量、进程列表、慢查询、锁信息等
   - 提示: 可以一次执行多条用分号分隔的查询
   - 常用查询示例:
     * SHOW GLOBAL STATUS LIKE 'Slow_queries'
     * SHOW VARIABLES LIKE 'long_query_time'
     ...
```

注意，我还特意加了"常用查询示例"。这很重要，因为 LLM 有时候不知道具体该用什么命令，给几个例子能帮它快速上手。

#### 第三部分：诊断流程建议

我还给了 LLM 一个诊断流程的建议，但强调"可以根据实际情况调整"：

```python
【诊断流程建议】

请按以下思路进行诊断，但可以根据实际情况调整：

第1阶段 - 基础检查（建议第1-3轮）：
1. 慢查询统计 - Slow_queries, long_query_time
2. MySQL 运行时间 - Uptime
3. 查询量统计 - Questions, 计算 QPS
...
```

这就像给新手医生一个检查清单——既提供指导，又不限制灵活处理。

#### 第四部分：响应格式

最后，必须明确告诉 LLM 如何输出它的决策：

```python
【响应格式】

你的回答必须是纯 JSON 格式，不要包含其他文字：

{
  "tool": "mysql_query|ssh_exec|final_answer",
  "action": "具体动作描述",
  "input": "要执行的查询或命令"
}
```

格式约定是 Agent 开发中最容易出问题的地方。LLM 有时候会在 JSON 外面加一些解释，或者用 Markdown 代码块包裹。所以在代码里，我加了一些清理逻辑：

```python
# 清理响应 - 有时会有额外的文本或 markdown 格式
assistant_message = assistant_message.strip()
if assistant_message.startswith("```json"):
    assistant_message = assistant_message[7:]
if assistant_message.startswith("```"):
    assistant_message = assistant_message[3:]
if assistant_message.endswith("```"):
    assistant_message = assistant_message[:-3]
assistant_message = assistant_message.strip()

parsed = json.loads(assistant_message)
```

这些细节虽然琐碎，但能让你的 Agent 稳定很多。

## 三、把它们拼起来：Agent 的主循环

现在三大组件都有了，怎么把它们拼起来呢？核心就是一个**主循环**：

```python
def run(self) -> str:
    """运行完整的诊断流程"""
    logger.info("诊断 Agent 启动...")
    self.current_round = 0

    # 初始上下文
    user_message = self._build_initial_context()

    while self.current_round < self.MAX_ROUNDS:
        self.current_round += 1
        logger.info(f"--- 第 {self.current_round} 轮诊断 ---")

        # 步骤 1：让 LLM 决定下一步
        tool_call = self._call_llm(user_message)
        logger.info(f"决策: {tool_call.tool} - {tool_call.action}")

        # 步骤 2：检查是否结束诊断
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

        # 步骤 3：执行工具
        observation = self._execute_tool(tool_call)

        # 步骤 4：记录观察
        self.report_writer.add_observation(
            tool=tool_call.tool,
            action=tool_call.action,
            input_data=tool_call.input_data,
            observation=observation,
            round_num=self.current_round,
        )

        # 步骤 5：准备下一轮消息
        user_message = f"""来自 {tool_call.tool}.{tool_call.action} 的观察结果:
...
"""
```

这个循环就是 Agent 的核心。每一轮都是：**观察 → 决策 → 行动 → 再观察**，直到 LLM 认为信息足够了。

## 四、实际开发中的经验和坑

开发这个 Agent 的过程中，我踩了一些坑，也总结了一些经验，跟你分享一下。

### 经验一：先验证，再执行

这是最重要的一条。无论是 SSH 命令还是 SQL 查询，**先验证安全性，再执行**。

我一开始也想图省事，觉得"LLM 应该不会执行危险命令"。但仔细想想，这是很危险的——LLM 有时候会产生幻觉，或者用户的问题会诱导它做一些不该做的事。

所以，**不要信任 LLM 的输出，所有的工具调用都必须经过验证**。

### 经验二：给 LLM 例子，比说一堆道理有用

在 Prompt 里，我花了很多篇幅写"常用查询示例"和"常用命令示例"。这比空泛地说"请检查慢查询"要有效得多。

LLM 虽然聪明，但有时候不知道具体的语法。给几个例子，能大大减少它"想当然"的情况。

### 经验三：处理好边界情况

实际运行时，你会遇到各种边界情况：
- LLM 返回的 JSON 格式不对
- 工具执行失败
- 网络超时
- LLM 陷入死循环，反复查同样的东西

我的建议是：
- 加一个最大轮次限制（我设的是 30）
- JSON 解析要有容错处理
- 工具执行失败也要记录下来，告诉 LLM
- 给 LLM 的观察结果要简洁明了，不要把几十兆的日志全丢给它

### 经验四：记录一切，便于调试

开发 Agent 的时候，你会经常想："它为什么会做这个决策？"

所以，**把一切都记录下来**：
- 每一轮的 LLM 输入和输出
- 工具的执行结果
- 最后的诊断结论

我的 ReportWriter 不仅是给用户看的，也是给开发者调试用的。有了完整的记录，你才能知道 Agent 为什么会这么想，进而优化 Prompt 或 Tools。

## 五、常见问题与解答

### Q1：LLM 会不会很贵？

A：这个项目用的是 DeepSeek-V3.2，价格很便宜。一次完整的诊断（约 20 轮）大概也就几分钱人民币。相比它节省的时间，这个成本完全可以忽略。

### Q2：LLM 做的决策不靠谱怎么办？

A：这很正常。我的建议是：
1. 在 Prompt 里给更明确的指导
2. 把 Tools 做得更强大，提供更多有用的信息
3. 可以在 Prompt 里要求 LLM 先思考再行动（"请先分析已有信息，再决定下一步"）

### Q3：这个 Agent 能替代 DBA 吗？

A：不能。它的定位是**辅助工具**，不是替代品。它能帮你完成繁琐的数据收集工作，但最终的判断和决策还是要人来做。

而且，目前这个版本还不能执行修复操作（因为是只读的），它只能告诉你问题在哪里，怎么修还得你来。

## 六、总结和展望

### 总结一下

今天我们聊了 Agent 开发的三大核心组件：

1. **LLM** - 做决策的大脑
2. **Tools** - 执行操作的手脚（这是 80% 的工作所在）
3. **Prompt** - 告诉 LLM 如何工作的说明书

开发 Agent 的秘诀是：**把 LLM 当作一个聪明但需要明确指导的实习生**。你要给它清晰的角色定位、好用的工具、明确的指令，然后让它自己去工作。

### 未来的改进方向

这个 Agent 还有很多可以改进的地方：

1. **更丰富的工具集**：可以加更多检查项，比如慢查询日志分析、表结构检查等
2. **知识库**：把常见的问题和解决方案做成知识库，让 LLM 参考
3. **历史对比**：对比历史诊断结果，发现趋势性问题
4. **修复建议的可操作性**：给出更具体的修复步骤，甚至可以（谨慎地）执行一些安全的修复操作

### 给你的建议

如果你也想开发自己的 Agent，我的建议是：

1. **从简单开始**：不要一开始就想做一个全知全能的 Agent，先解决一个具体的小问题
2. **重点写 Tools**：把 80% 的精力放在 Tools 上，让它们可靠、好用
3. **快速迭代**：Prompt 很难一次写好，要用实际运行的结果来不断优化
4. **安全第一**：如果涉及实际操作，一定要有严格的安全边界

## 最后

AI 工具不是魔法，它背后是清晰的逻辑和扎实的代码。理解了这些原理，你不仅能更好地使用现有的 AI 工具，还能开发出自己的 Agent。

希望这篇文章对你有帮助。如果你也开发了有趣的 Agent，欢迎在评论区分享！

---

*本文基于一个真实的 MySQL/Linux 诊断 Agent 项目编写。如果你对代码感兴趣，可以查看项目源码。*
