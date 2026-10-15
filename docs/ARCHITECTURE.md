# 实现架构

模块化 Go 单体。CLI、HTTP 与 MCP 共用一个应用服务和索引，React 网页仅负责交互。

| 文件 | 职责 |
|---|---|
| `model.go` | 版本化对象、本机配置、校验值、原子持久化 |
| `sources.go` | 原生记录快照、三个来源解析、附件、监听、补扫 |
| `store.go` | 封存记录、回放、消息版本、记忆 DAG、上下文、任务认领 |
| `search.go` | 中文分词、FTS 身份映射与可重建时间顺序 |
| `transfer.go` | 自包含导出、校验导入、离线暂存、合并、遗忘、接续 |
| `integrations.go` | 配置合并、平台事件、插件、用户级启动服务 |
| `server.go` | 回环 HTTP、鉴权、来源校验、SSE、持久化事件队列 |
| `mcp.go` | JSON-RPC stdio 与六项业务工具 |
| `cli.go` | 初始化、查询、迁移和生命周期命令 |
| `web/` | 四页界面、Markdown 阅读、编辑、依据定位 |

## 持久化顺序

读取来源时先生成同一份不可变快照，原始字节和解析都使用该快照，避免读取活跃文件的两个不同状态。原始内容分块保存并校验；会话变化写入设备独立的封存段并 fsync，随后提交 SQLite 事务，最后推进本机采集位置。索引失败不会回滚原档。部分尾部保留原始字节，下次活动重试。

OpenCode 通过 `mode=ro` 连接与只读事务获取 session/message/part 原生行，保存对应会话的快照；不复制活跃数据库文件。未知字段随原生资料保留。Codex 与 Claude 的规范化阅读视图只呈现可识别的可见消息，完整工具调用等仍保留在原生档案中。无法取得的附件标记缺失。

消息身份取原生 ID；Codex 旧格式缺少消息 ID 时优先用 turn 身份与轮内序号，最后才退回原始行序号。原生修订保留所有事件版本。对于没有任何稳定身份和顺序线索的未知来源格式，原生字节仍可保留，但无法保证自动推断编辑/分支关系。

## 多机器与版本

设备标识只存在本机配置。共享目录没有活跃数据库，各设备以独立路径写入完整封存文件；同一载荷有相同逻辑记录 ID，因此重扫与重复导入不增加消息。新文件回放前先处理删除标记，避免到达顺序使已遗忘内容复活。

记忆以父版本形成 DAG。不同 agent 基于同一父版本写入时，两条分支同时有效；当前 agent 阅读后用多个 parents 合并。Markdown 首行保存版本依据，手工正文编辑生成新版本。

单机服务有进程锁，原始对象归档、导入、导出与遗忘序列化；索引事务有边界。维护任务有五分钟租约，可重复认领自己的任务，其他 agent 等待租约到期。原生消息与记忆的旧版本在封存段中，当前索引只是阅读投影。

## 查询

中文单字和双字、英文词元使用 FTS5。查询多字中文时使用双字交集，避免重复单字约束。默认按来源秒级时间倒序，同秒用确定性的身份后缀排序，完整保留分页；重建后保持时间顺序。可选 BM25 相关度排序。独立的身份映射表保证更新一条流式回复时不会扫描全库。

启动上下文包含协议、基础记忆、当前摘要、预算内的未整理尾部、相关片段及少量维护任务。完整原文与消息版本通过分页读取，不因上下文预算截断档案。摘要未覆盖内容仍列为维护事项。

## 本机边界

HTTP 仅监听回环 IP，验证精确 Host 和 Origin；API 需要本机凭据，网页使用 HttpOnly、SameSite=Strict cookie。前端不执行原始 HTML，CSP 禁止档案内脚本和远程加载。访问凭据、配置副本、日志、队列和 SQLite 留在本机应用目录。

默认配置位于系统用户配置目录下的 `memvault/`。macOS 是 `~/Library/Application Support/memvault`，Linux 使用用户配置目录，Windows 使用用户配置目录；可由 `MEMVAULT_HOME` 指定。

## 主要接口

本机 HTTP 前缀为 `/api/v1/`：`status`、`sessions`、`sessions/{id}`、`search`、`context`、`memories`、`tasks`、`relations`、`capture`、`scan`、`events`、`export`、`import`、`forget/preview`、`forget`、`resume`、`settings`、`relocate`、`repair`。

所有业务接口需鉴权，变更接口只允许 POST。`GET sessions/{id}?message=消息编号` 定位依据；`?format=markdown` 导出阅读原文。`repair` 的动作是 `verify`、`rebuild`、`integrations`。

格式兼容边界与原生能力参考：[Codex hooks](https://learn.chatgpt.com/docs/hooks)、[Claude Code hooks](https://code.claude.com/docs/en/hooks)、[OpenCode plugins](https://docs.opencode.ai/docs/plugins/)、[SQLite FTS5](https://sqlite.org/fts5.html)、[SQLite WAL](https://sqlite.org/wal.html)。
