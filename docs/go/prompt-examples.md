# Prompt Examples / 提示词示例

## 中文版

### 这份文档适合谁

- 想直接复制提示词来使用 mem 的普通用户
- 想知道“应该怎么对 Codex 开口”的协作者

### 这份文档不解决什么问题

- 不解释 Go 实现细节
- 不解释 MCP 传输协议
- 不替代 [how-memory-works.md](./how-memory-works.md) 的概念说明

### 目的

这份文档给你一些可以直接发给 Codex 的提示词示例，帮助它正确使用 `codex-mem`。

这些不是 MCP 方法名，也不是命令行。
它们是 `codex-mem` 已经注册成 MCP server 之后，用户直接发给 Codex 的自然语言。

### 先记住一件事

大多数情况下，你不用手动指定 `scope`。

更实用的理解是：

- 先让系统识别“你现在在哪个仓库/项目里”
- 再开始会话、恢复上下文
- 后续保存记忆时，默认就会保存到当前仓库/项目下面

只有在这些情况，才值得明确提“范围”：

- 当前目录不是标准仓库，怕识别不准
- 你想跨项目查记忆
- 你有多个 clone / worktree，想先确认系统识别到了哪一个

### 如何理解这些示例

- 你只需要写自然语言
- Codex 会自己决定调用哪个 MCP 工具
- 下方列出的工具名只是帮助你理解它背后会做什么，不是让你手动输入

### 开始工作

#### 提示词：先恢复这个仓库之前的上下文，再继续当前任务

```text
先恢复这个仓库之前的上下文。如果有相关的交接记录或长期笔记，就一起加载，然后开始一个新的会话继续当前任务。
```

可能调用的工具:

- `memory_bootstrap_session`

#### 提示词：如果仓库识别可能不准，先确认当前仓库/项目，再开始会话

```text
先确认一下当前工作目录属于哪个仓库和项目，再开始一个新的带记忆会话。
```

可能调用的工具:

- `memory_resolve_scope`
- `memory_start_session`

### 保存长期有效的信息

#### 提示词：把这次结论保存成长期记录

```text
这个结论以后还会反复用到，请把它保存成长期记忆，重要度高一点，并带上相关文件。
```

可能调用的工具:

- `memory_save_note`

#### 提示词：把这次 bug 修复经验保存下来

```text
这次 bug 的根因和修复方法以后可能还会用到，请整理成一条长期记忆，顺便带上改过的文件。
```

可能调用的工具:

- `memory_save_note`

### 保存交接记录

#### 提示词：结束前保存交接记录

```text
结束之前请保存一条交接记录，写清楚这次做了什么、下一步做什么、还有哪些问题没解决，以及改过哪些文件。
```

可能调用的工具:

- `memory_save_handoff`

#### 提示词：中途先存一个检查点

```text
先为当前任务保存一个检查点，方便我下次从现在这个状态继续。
```

可能调用的工具:

- `memory_save_handoff`

### 查历史信息

#### 提示词：搜索以前的记忆

```text
先搜索一下以前和 HTTP MCP 传输、origin 校验或者远程部署相关的历史笔记和交接记录。
```

可能调用的工具:

- `memory_search`

#### 提示词：当前在 A 项目开发，但要参考 B 项目的历史经验

```text
我现在在 A 项目里开发，这次任务和 B 项目有关。请搜索一下 B 项目里与当前任务相关的历史笔记或交接记录，并明确标出哪些结果来自 B 项目。
```

可能调用的工具:

- `memory_search`

#### 提示词：先看最近的上下文

```text
继续之前，先把这个仓库最近的笔记和交接记录加载出来给我看。
```

可能调用的工具:

- `memory_get_recent`

#### 提示词：开场时就一起加载相关项目上下文

```text
先恢复当前 A 仓库之前的上下文。这次任务依赖 B 项目，也一起加载和 B 项目相关的历史记忆，并标明哪些内容来自 B 项目。
```

可能调用的工具:

- `memory_bootstrap_session`

#### 提示词：读取一条完整记录

```text
把这条记忆记录的完整内容加载出来给我看。
```

可能调用的工具:

- `memory_get_note`

### 安装 AGENTS 指引

#### 提示词：给当前仓库安装推荐指引

```text
请为当前仓库以 safe 模式安装推荐的 AGENTS.md 指引。
```

可能调用的工具:

- `memory_install_agents`

### 更容易触发 memory 的表达

这些表达更容易让 Codex 明确知道你是在要求它使用持久记忆：

- `长期记录`
- `保存到记忆`
- `持久化保存`
- `交接记录`
- `下次继续`
- `恢复上下文`
- `搜索记忆`
- `最近笔记`

### 不够稳定的表达

这些说法太模糊，Codex 不一定会稳定调用 memory 工具：

- `记一下`
- `你记住这个`
- `先放脑子里`

如果你真的希望它进入持久记忆，最好直接说：

- `把这个保存成长期记录`
- `请保存一条交接记录`
- `请先恢复这个仓库之前的上下文`

## English Version

### Purpose

This document gives example user prompts that encourage Codex to use `codex-mem` correctly.

These are not MCP method calls.
They are normal prompts a user can send to Codex after `codex-mem` has already been registered as an MCP server.

### One Practical Rule

In most cases, you do not need to specify `scope` manually.

The practical model is:

- let the system identify the current repository/project first
- start or bootstrap the session
- save notes and handoffs into that current repository/project context

You only need to be more explicit when:

- the current directory may be ambiguous
- you want cross-project retrieval
- you have multiple clones or worktrees and want to confirm what was identified

### How To Read These Examples

- the user writes the prompt in natural language
- Codex decides which MCP tool or tools to call
- the tool names below are the likely tools Codex should choose, not text the user needs to type

### Session Start

#### Prompt: recover prior context and continue work

```text
Recover prior context for this repository. If there are relevant handoffs or durable notes, load them and start a new session so we can continue the current task.
```

Likely tools:

- `memory_bootstrap_session`

#### Prompt: confirm the current repository/project first

```text
First confirm which repository and project this working directory belongs to, then start a new memory-backed session.
```

Likely tools:

- `memory_resolve_scope`
- `memory_start_session`

### Saving Durable Discoveries

#### Prompt: save this as a durable record

```text
This is likely to matter again later. Save it as a durable memory note with high importance and include the relevant files.
```

Likely tools:

- `memory_save_note`

#### Prompt: save a reusable bugfix insight

```text
This bug's root cause and fix may be useful again later. Save a durable note with the summary and the touched files.
```

Likely tools:

- `memory_save_note`

### Handoffs

#### Prompt: save a handoff before stopping

```text
Before stopping, save a handoff that summarizes what was done, what should happen next, what is still unresolved, and which files were changed.
```

Likely tools:

- `memory_save_handoff`

#### Prompt: save a checkpoint mid-task

```text
Save a checkpoint for this task so I can resume from the current state later.
```

Likely tools:

- `memory_save_handoff`

### Retrieval

#### Prompt: search prior memory

```text
Search prior notes and handoffs related to HTTP MCP transport, origin checks, or remote deployment.
```

Likely tools:

- `memory_search`

#### Prompt: work in project A while referencing project B

```text
I am working in project A, and this task depends on project B. Search for prior notes or handoffs from project B that are relevant to the current task, and clearly label which results come from project B.
```

Likely tools:

- `memory_search`

#### Prompt: show the latest context first

```text
Before we continue, show the most recent notes and handoffs for this repository.
```

Likely tools:

- `memory_get_recent`

#### Prompt: load related-project context at startup

```text
Recover prior context for the current A repository. This task depends on project B as well, so also load relevant memory from project B and clearly label which items come from project B.
```

Likely tools:

- `memory_bootstrap_session`

#### Prompt: fetch one exact record

```text
Load the full content of this saved memory record for me.
```

Likely tools:

- `memory_get_note`

### AGENTS Installation

#### Prompt: install workflow guidance

```text
Install the recommended AGENTS.md guidance for this repository in safe mode.
```

Likely tools:

- `memory_install_agents`

### Stronger Prompt Patterns

Prompts work better when they include one or more of these signals:

- `durable`
- `save to memory`
- `handoff`
- `resume later`
- `recover prior context`
- `search memory`
- `recent notes`

### Weak Prompt Patterns

These are more ambiguous and may not reliably cause Codex to use memory:

- `remember this`
- `keep this in mind`
- `note this somewhere`

If durable memory matters, say that explicitly:

- `save this as a durable record`
- `please save a handoff`
- `please recover prior context for this repository first`
