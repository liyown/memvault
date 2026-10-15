# Memvault

<!-- impeccable:product-schema 1 -->

## Platform
web

## Stack
用户确认：Go 后端、React + TypeScript 本地网页、编译资源嵌入单个程序；CLI、HTTP、MCP 共用应用服务。

## Users
单个用户跨 Codex、Claude Code、OpenCode 和不同机器工作。日常不应手动整理记忆。

## Product Purpose
完整会话与长期记忆作为用户持有的本地资产。平台记录消失后，已归档资料仍可读取、检索、导出、用于其他 agent 接续。

## Operating Context
用户指定本文件夹作为记忆位置，自行负责备份和跨机器同步。后台服务与 hook 自动采集；当前 agent 整理记忆，不额外调用模型。

## Capabilities and Constraints
四个导航：会话、记忆、接入、设置。会话平级保存，摘要和对话分开阅读，记录关联与归档范围。跨 agent 采用摘要接续＋完整原文可查。索引和本机凭据不进入共享记忆目录。

## Evidence on Hand
已有本机 Codex、OpenCode 来源；Claude Code 有本地历史但无可执行命令。性能和跨平台验证在完成前不得宣称通过。

## Product Principles
- 原文先归档，解释和摘要随后处理。
- 保留版本、证据和冲突，不让后台判断用户事实。
- 配置合并，不改变无关权限。
- 正常安静，异常准确说明影响与恢复方式。
- 数据与工具生命周期独立。
