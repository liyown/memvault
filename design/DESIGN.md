---
name: Memvault · 档案阅读室
description: 清晰、舒适，适合本地会话与记忆的长文阅读。
colors:
  canvas: "#f4f3ee"
  paper: "#fffefb"
  sunken: "#ecebe4"
  hover: "#e9e8e0"
  ink: "#1f2420"
  prose: "#3c433d"
  muted: "#626b5d"
  faint: "#8c9387"
  line: "#e4e3da"
  line-strong: "#d2d3c7"
  accent: "#36533f"
  accent-hover: "#2a4532"
  accent-soft: "#e5ece1"
  good: "#3b7149"
  warning: "#87562a"
  danger: "#a33d37"
typography:
  page-title:
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif'
    fontSize: "24px"
    fontWeight: 650
    letterSpacing: "-0.02em"
  reader-title:
    fontSize: "21px"
    fontWeight: 650
    lineHeight: 1.4
  directory-title:
    fontSize: "19px"
    fontWeight: 650
  body:
    fontSize: "15.5px"
    lineHeight: 1.85
  body-mobile:
    fontSize: "15px"
  ui:
    fontSize: "13–14px"
  meta:
    fontSize: "12–12.5px"
  code:
    fontFamily: 'ui-monospace, "SF Mono", SFMono-Regular, Menlo, Consolas, monospace'
    fontSize: "12–12.5px"
rounded:
  xs: "4px"   # 徽章、kbd、头像
  sm: "6px"   # 按钮、输入、列表行、代码块
  md: "10px"  # 卡片、工作区、弹出菜单、提示
  lg: "14px"  # 对话框
spacing: "4 / 8 / 12 / 16 / 20 / 24 / 32 / 40px"
controls:
  height: "36px"
  compact: "30px"
---

# Design System: Memvault · 档案阅读室

## Overview

**Creative North Star: "档案阅读室"**

把本地会话作为可以连续阅读、查证与接续的档案。导航放在暖灰画布上，工作区是一张浮起的纸面，内容始终是视觉主体。森林绿只用于主要操作、选中与焦点。界面安静、紧凑、长文舒适，属于 Operate / Read 的工作环境。

**Key Characteristics:**

- 两层表面：画布（canvas）承载导航，纸面（paper）承载工作内容。工作区是唯一常驻带阴影的面。
- 圆角只有四档（4 / 6 / 10 / 14px），全部来自 token。
- 一套状态语汇：圆点＋文字，三种语气（good / warning / neutral），不用颜色单独表达状态。
- 每个视图最多一个主按钮；其余是描边、ghost 或图标按钮。
- 信息只出现一次：不在页脚、顶栏和侧栏重复“本地运行”之类的口号。

## Colors

颜色值以前置 tokens 为准，代码中的唯一来源是 `web/src/styles/base.css` 的 `:root`。

- **canvas / paper / sunken / hover**：画布、纸面、凹陷底（代码块、分段控件、徽章）、悬停。
- **ink / prose / muted / faint**：标题与控件、长文、说明、元数据四级文字。
- **line / line-strong**：结构分隔与控件边框。
- **accent / accent-soft**：主按钮、选中行、焦点；accent-soft 用于选中底色与提示。
- **good / warning / danger** 及对应 soft：状态点、横幅、危险操作。

来源标识（Codex / Claude Code / OpenCode）使用 18px 方形字母标，颜色只作区分辅助，旁边总有名称。

## Typography

系统中文无衬线栈；代码与路径使用等宽字体。界面文字（13–14px）与阅读正文（15.5px / 1.85）分开定义，正文最大行长 72ch。

- 页面标题 24px，阅读区标题 21px，目录标题 19px，卡片/段落标题 14.5px。
- 元数据 12–12.5px，等宽路径 12.5px。
- 数字使用 tabular-nums。

## Layout

- **外壳**：侧栏 232px（≤1100px 为 200px，可收为 64px 图标轨道），工作区四周留 8px 浮在画布上，圆角 10px。
- **会话 / 记忆**：目录 336px（≥1500px 为 360px，≤1100px 为 280px）＋ 阅读区。目录头部承载页面标题、计数和图标操作，不另设页面标题行。阅读区头部包含来源·时间·条数、标题、工具条和标签页；正文在 72ch 版心内居中。
- **接入 / 设置**：居中最大 960px 的单列页面，PageHeader（标题＋说明＋右侧操作）在上。设置每段为“左说明列 220px ＋ 右卡片”，卡片底部为操作栏；≤1100px 转单列。
- **≤800px**：侧栏变为抽屉（遮罩、焦点限制、Escape 返回、关闭时 inert），工作区铺满并出现 52px 顶栏。
- **≤640px**：目录与阅读区切换显示，阅读区顶部有“返回列表”。

## Elevation & Depth

工作区使用 `--shadow-raised`，弹出菜单、对话框、提示使用 `--shadow-pop`。卡片、列表、表格只用细线分隔，不加阴影。

## Components

共用组件位于 `web/src/ui.tsx` 与 `web/src/path-picker.tsx`：

- **PageHeader**：标题、一行说明、右侧操作组。
- **Button**：primary / 默认描边 / ghost / icon（30px 方形）/ danger；高度 36px。AsyncButton 忙碌时保持宽度并禁用。
- **Field**：标签在上（13px/600），说明与错误在下；输入高 36px；`input-affix` 用于单位后缀。
- **SelectField**：原生 select，统一收起态与箭头；`compact` 变体 30px，用于筛选。
- **Segmented**：分类筛选，替代散落的筛选按钮。
- **Switch**：30×18 开关，与名称同在一个 label 内。
- **StatusBadge / flag**：状态点＋文字；小号 flag 用于“待确认”“附件待补齐”等行内标记。
- **ListRow**（`.list-row`）：会话与记忆共用。来源·时间一行、标题两行截断、摘要两行截断；选中为 accent-soft 底＋左 2px 标线。
- **PathField**：等宽路径显示，长路径在非编辑态中间省略，聚焦后显示完整可编辑；右侧复制与“选择…”。“选择…”请求本机服务打开系统原生对话框（macOS osascript、Windows PowerShell、Linux zenity/kdialog），不可用时打开应用内 FolderBrowser。
- **FolderBrowser**：原生 dialog；上一级、个人文件夹、可点击路径分段、键盘上下选择、显示隐藏项目、底部显示目标路径并确认。关闭后焦点回到触发按钮。
- **Dialog**：14px 圆角，头部图标/标题＋关闭，底部右对齐操作。遗忘对话框先展示影响范围。
- **Empty**：图标块＋标题＋说明，可附操作。

### Messages

消息按文档段落排列，作者行（18px 标识、名称、时间、悬停出现的复制）在上，没有聊天气泡。工具输出（含 Claude Code 以用户回合保存的“工具结果”）默认折叠为一行等宽摘要，展开后以等宽块显示并限制高度。Markdown 支持 GFM 表格、任务列表与删除线，不执行原始 HTML。

### Motion

状态切换 140ms；面板、抽屉、侧栏宽度 240ms，cubic-bezier(0.16, 1, 0.3, 1)；弹出层使用短暂的 pop-in。prefers-reduced-motion 时关闭过渡、动画和平滑滚动。

## Do's and Don'ts

### Do:

- **Do** 从 token 取圆角、颜色和尺寸；新组件先查 `ui.tsx` 是否已有。
- **Do** 让路径可选择、可复制、可编辑，三种方式都在同一个 PathField 里。
- **Do** 在未保存修改时提示并把保存按钮升为主按钮；无修改时禁用。
- **Do** 让状态反映来源实际范围，用文字解释异常。

### Don't:

- **Don't** 在同一视图放多个主按钮，或在一行里混用多种按钮层级。
- **Don't** 重复展示同一信息（总数、本地状态、口号）。
- **Don't** 直接显示原生会话 id 或包装标签作为标题；交给 `sessionTitle`。
- **Don't** 新增硬编码圆角或颜色。
