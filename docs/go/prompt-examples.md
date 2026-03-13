# Prompt Examples / 提示词示例

## Purpose / 目的

This document gives example user prompts that encourage Codex to use `codex-mem` correctly.

本文给出一些用户可直接使用的提示词示例，帮助 Codex 正确调用 `codex-mem`。

These are not MCP method calls.
They are normal prompts a user can send to Codex after `codex-mem` has already been registered as an MCP server.

这些不是 MCP 方法名，也不是命令行。
它们是用户在 `codex-mem` 已注册为 MCP server 之后，直接发给 Codex 的自然语言提示词。

## How To Read These Examples / 如何理解这些示例

- the user writes the prompt in natural language
- Codex decides which MCP tool or tools to call
- the tool names below are the likely tools Codex should choose, not text the user needs to type

- 用户只需要写自然语言提示词
- Codex 会自己决定调用哪个 MCP 工具
- 下方列出的工具名只是“Codex 可能会选用的工具”，不是用户需要手工输入的内容

## Session Start / 启动与恢复

### Prompt / 提示词: recover prior context and start work / 恢复上下文并开始工作

```text
先恢复当前仓库之前的上下文，如果有相关的交接记录或记忆笔记就一起加载，然后开始一个新的会话来继续当前任务。
```

```text
Recover prior context for this repository, load any relevant handoff or notes, and start a new session to continue the current task.
```

Likely tools / 可能调用的工具:

- `memory_bootstrap_session`

### Prompt / 提示词: resolve scope first, then start / 先解析 scope，再启动会话

```text
先解析当前工作区的 scope，再为这个仓库开启一个新的带记忆的会话。
```

```text
Resolve the current workspace scope first, then start a fresh memory-backed session for this repository.
```

Likely tools / 可能调用的工具:

- `memory_resolve_scope`
- `memory_start_session`

## Saving Durable Discoveries / 保存长期有效的信息

### Prompt / 提示词: save a durable note / 保存长期记录

```text
这个结论是长期有效的，请把它保存成一条记忆笔记，重要度设高一点，并带上受影响的文件。
```

```text
This decision is durable. Save it as a memory note with high importance and include the affected files.
```

Likely tools / 可能调用的工具:

- `memory_save_note`

### Prompt / 提示词: save a reusable bugfix insight / 保存可复用的 bug 修复经验

```text
我们刚刚得到一个可复用的 bug 修复经验，请把根因、修复方式和改动过的文件保存成长期记忆笔记。
```

```text
We found a reusable bugfix insight. Save a durable memory note summarizing the root cause, fix, and touched files.
```

Likely tools / 可能调用的工具:

- `memory_save_note`

## Handoffs / 交接记录

### Prompt / 提示词: save a continuation handoff before stopping / 结束前保存交接记录

```text
在结束之前，请保存一条交接记录，写清楚这次做了什么、下一步做什么、还有哪些未解决问题，以及改过哪些文件。
```

```text
Before ending, save a handoff for the next session with a summary, completed work, next steps, open questions, and touched files.
```

Likely tools / 可能调用的工具:

- `memory_save_handoff`

### Prompt / 提示词: checkpoint mid-task / 中途保存检查点

```text
先为当前任务保存一条检查点交接记录，方便之后从当前状态继续。
```

```text
Create a checkpoint handoff for this task so we can resume later from the current state.
```

Likely tools / 可能调用的工具:

- `memory_save_handoff`

## Retrieval / 检索历史信息

### Prompt / 提示词: search prior memory / 搜索历史记忆

```text
先搜索一下 memory 里和 HTTP MCP 传输、origin 校验或远程部署相关的历史笔记和交接记录。
```

```text
Search memory for prior notes or handoffs related to HTTP MCP transport, origin checks, or remote deployment.
```

Likely tools / 可能调用的工具:

- `memory_search`

### Prompt / 提示词: load recent context without a query / 不带搜索词加载最近上下文

```text
继续之前，先把这个仓库最近的笔记和交接记录加载出来给我看。
```

```text
Show the most recent notes and handoffs for this repository before we continue.
```

Likely tools / 可能调用的工具:

- `memory_get_recent`

### Prompt / 提示词: fetch one exact record / 读取一条完整记录

```text
把这条记忆记录的完整内容加载出来给我。
```

```text
Load the full memory record for this note id and show the complete saved content.
```

Likely tools / 可能调用的工具:

- `memory_get_note`

## AGENTS Installation / 安装 AGENTS 指引

### Prompt / 提示词: install workflow guidance / 安装工作流指引

```text
请为当前仓库以 safe 模式安装推荐的 AGENTS.md 指引。
```

```text
Install the recommended AGENTS.md guidance for this repository in safe mode.
```

Likely tools / 可能调用的工具:

- `memory_install_agents`

## Stronger Prompt Patterns / 更容易触发 memory 的表达

Prompts work better when they include one or more of these signals:

这些词更容易让 Codex 明确判断你是在要求它使用持久记忆：

- `长期记录`
- `保存到记忆`
- `持久化保存`
- `交接记录`
- `下次继续`
- `恢复上下文`
- `搜索记忆`
- `最近笔记`

English equivalents:

- `durable`
- `save to memory`
- `handoff`
- `resume later`
- `bootstrap context`
- `search memory`
- `recent notes`

## Weak Prompt Patterns / 不够稳定的表达

These are more ambiguous and may not reliably cause Codex to use memory:

这些说法太模糊，Codex 不一定会稳定调用 memory 工具：

- `记一下`
- `你记住这个`
- `先放脑子里`
- `remember this`
- `keep this in mind`
- `note this somewhere`

If durable memory matters, say that explicitly.

如果你真的希望它进入持久记忆，最好明确说：

- `把这个保存成长期记录`
- `请保存一条交接记录`
- `请先恢复这个仓库之前的上下文`
