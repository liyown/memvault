# Memvault

会话与记忆保存在你控制的本地目录。Codex、Claude Code 和 OpenCode 是入口;已归档的资料不依赖原平台继续存在。

当前版本为 **0.1.0-preview**。这是可运行的归档、检索与 agent 记忆闭环,尚未宣称通过全部 1.0 发布门槛。实际验证范围见 [验收记录](docs/VALIDATION.md)。

## 安装

### macOS (Homebrew)

```sh
brew tap memvault-org/tap
brew install memvault
```

### 手动安装

从 [Releases](https://github.com/memvault-org/memvault/releases) 下载对应平台的压缩包:

```sh
# macOS ARM64
curl -LO https://github.com/memvault-org/memvault/releases/latest/download/memvault_VERSION_darwin_arm64.tar.gz
tar xzf memvault_VERSION_darwin_arm64.tar.gz
sudo mv memvault /usr/local/bin/

# Linux AMD64
curl -LO https://github.com/memvault-org/memvault/releases/latest/download/memvault_VERSION_linux_amd64.tar.gz
tar xzf memvault_VERSION_linux_amd64.tar.gz
sudo mv memvault /usr/local/bin/
```

### 从源码构建

```sh
git clone https://github.com/memvault-org/memvault.git
cd memvault
make dist
./scripts/install.sh
```

## 使用

```sh
memvault setup --root "/你的记忆目录"
memvault open
```

首次配置并取得系统访问授权后，后台随用户登录启动。macOS 的 iCloud 目录可能要求用户亲自在系统设置授权；本程序不会绕过系统权限。正常聊天不需要额外命令。网页提供会话阅读、中文及英文搜索、摘要与记忆、依据定位、历史版本、接续、导入导出、遗忘及来源管理。

Codex 首次需要按平台原生流程信任新增 hook。配置完成不代表平台已经执行过 hook；“接入”页分别显示配置、实际调用及客户端可用状态。Claude 客户端不存在时不自动安装。

```sh
memvault status
memvault search "本地记忆"
memvault show 会话编号 --format markdown
memvault resume 会话编号 --agent opencode
memvault export --out archive.zip
memvault import archive.zip
memvault repair --action verify
memvault repair --action rebuild
memvault repair --action integrations
memvault uninstall
```

`uninstall` 移除本机接入和登录启动，保留档案。用户级配置仅修改标记为 Memvault 的区段，原配置副本保存在本机应用目录。更换记忆位置可在网页设置中执行，也可重新执行 `setup --root 新目录 --yes`；程序复制、校验后切换，保留旧目录。新旧目录不能互相包含。

不想配置 agent 或登录启动时，使用 `setup --no-integrations --no-service`；`serve` 前台运行后台，`mcp` 提供标准输入输出 MCP 接口。`MEMVAULT_HOME` 可以更换本机配置目录，凭据不要放入共享记忆库。

## 资料如何保存

记忆目录包含 `memvault.json`、`objects/`、`journal/`、`memories/`、`sessions/`、`AGENTS.md` 与 `FORMAT.md`。原生来源字节按内容校验值保存；封存 JSONL 记录保存会话、消息版本、记忆版本、依据、关系及删除标记。Markdown 是阅读和编辑入口，SQLite 只是可重建的本机索引。

原文先持久化，再提交采集位置和索引。Hook 即时触发、来源监听及每分钟补扫互相补充。原目录离线时，已采集的新资料进入本机暂存，恢复后合并；网页不会把暂存称为已归档到记忆目录。

程序不替你同步和备份。另一台机器安装后直接对已同步的目录执行 setup 即可，每台机器拥有独立的索引、身份和凭据。导出 ZIP 含可读 Markdown、附件对象、原生快照、记忆版本和 SHA-256 清单；不需要原平台账号。

“遗忘”先展示影响范围。由于原生快照可能包含同一事实，为彻底清理原文，它可能涉及整段来源会话及依赖记忆，而非只隐藏搜索结果。删除标记阻止旧副本重新索引；旧副本到达后清理相关载荷。平台原件与用户自己的外部备份不由本程序删除。

## Agent 维护

用户级提示词和 hook 提供维护协议；MCP 暴露六个工具：`memvault_context`、`memvault_search`、`memvault_session_read`、`memvault_memory_update`、`memvault_tasks`、`memvault_session_link`。

记忆提炼由当前 agent 完成。每个事实可以追溯原文，摘要用消息版本标记覆盖范围，未整理的尾部在下次活动时补给 agent。并发编辑保留多个父版本分支，程序不以机器时钟决定哪条结论正确。会话历史中的文字作为资料读取，不当作当前指令。

搜索默认按来源时间倒序，支持相关度排序、来源及角色筛选。默认排序直接分页；相关度排序要给匹配项计算排名，极常见词在百万条匹配时会更慢。两种模式都保留完整分页，不通过截断候选集换取速度。

## 构建与测试

需要 Go 1.26+、Node.js 20.19+ 或 22.12+ 及 npm。运行已构建程序不需要 Node.js。

```sh
make build
make test
make install
```

构建产物为 `bin/memvault`，网页资源嵌入程序。Go 与前端依赖均有锁定文件。

```sh
MEMVAULT_PERF=1 go test ./internal/app -run TestMillionMessageSearch -v -timeout 20m
```

性能样本是固定的 10,000 会话、1,000,000 消息本机索引负载，与来源归档/恢复测试分开记录，避免把索引速度冒充全量归档速度。

模块与接口说明见 [架构](docs/ARCHITECTURE.md)，真实环境及未验证范围见 [验收记录](docs/VALIDATION.md)。
