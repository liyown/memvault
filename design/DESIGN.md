---
name: Memvault · 档案阅读室
description: 清晰、舒适，适合本地会话与记忆的长文阅读。
colors:
  paper: "#fffefa"
  page-bg: "#f5f5f0"
  sidebar: "#efefe8"
  ink: "#292e29"
  prose: "#454c40"
  muted: "#59654f"
  line: "#e3e4da"
  accent: "#3d5a46"
  accent-hover: "#304b39"
  selection: "#eaf0e4"
  nav-selection: "#dfe6d8"
  warning: "#936438"
  danger: "#a33d37"
typography:
  headline:
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif'
    fontSize: "27px"
    fontWeight: 650
    letterSpacing: "-0.025em"
  title:
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif'
    fontSize: "19px"
    fontWeight: 600
    lineHeight: 1.65
  body:
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif'
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.9
  body-mobile:
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif'
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.9
  label:
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif'
    fontSize: "12px"
  code:
    fontFamily: "ui-monospace, SFMono-Regular, Consolas, monospace"
    fontSize: "12px"
    lineHeight: 1.7
rounded:
  control: "7px"
  field-group: "8px"
  workspace: "10px 10px 0 0"
  dialog: "12px"
spacing:
  compact: "8px"
  small: "12px"
  regular: "16px"
  section: "24px"
  page: "32px"
components:
  button-primary:
    backgroundColor: "{colors.accent}"
    textColor: "#fff"
    rounded: "{rounded.control}"
    padding: "8px 12px"
  button-primary-hover:
    backgroundColor: "{colors.accent-hover}"
    textColor: "#fff"
  button-secondary:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    rounded: "{rounded.control}"
    padding: "8px 12px"
  button-danger:
    backgroundColor: "{colors.danger}"
    textColor: "#fff"
    rounded: "{rounded.control}"
    padding: "8px 12px"
  field:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    rounded: "{rounded.control}"
    padding: "9px 11px"
  navigation-active:
    backgroundColor: "{colors.nav-selection}"
    textColor: "#354d38"
    rounded: "{rounded.control}"
    padding: "11px 15px"
  reading-workspace:
    backgroundColor: "{colors.paper}"
    rounded: "{rounded.workspace}"
---

# Design System: Memvault · 档案阅读室

## Overview

**Creative North Star: "档案阅读室"**

把本地会话作为可以连续阅读、查证与接续的档案。暖白纸面承载正文，灰绿目录提供位置感，森林绿只在操作和选择处形成清晰信号。界面以安静、清楚和长文舒适度为先，属于 Operate / Read 的工作环境。

字形沿用系统中文无衬线字体。目录、时间与来源收敛为紧凑的辅助层，正文保留更宽松的字号和行距。结构依靠相邻色面、细边线和稳定对齐成立；持久导航贯穿会话、记忆、接入、设置。

**Key Characteristics:**

- 暖白正文、灰绿目录、墨色文字、森林绿操作。
- 目录密度与正文舒适度分别处理。
- 平面分区、细线与小圆角；浮层才使用阴影。
- 桌面并排查阅；手机会话在目录与正文之间切换。

## Colors

采用克制配色：纸面与灰绿中性色铺满界面，森林绿承担操作和选中信号。颜色值以前置 tokens 为准。

### Primary

- **森林绿**（accent / accent-hover）：主要操作及悬停，如继续会话、保存设置。
- **浅叶绿**（selection / nav-selection）：目录选中与当前导航，配合字重或边线表达位置。

### Neutral

- **暖白纸面**（paper）：原文、记忆正文、表单和阅读容器。
- **轻灰画布**（page-bg）与**灰绿目录**（sidebar）：工作背景与持久导航。
- **墨色**（ink）与**正文灰绿**（prose）：标题、控件及长文的不同阅读层。
- **次级灰绿**（muted）：来源、时间、说明与元数据；保持清楚可读。
- **浅灰绿边线**（line）：相邻内容、分栏和设置段落的结构界线。

### Functional States

- **赭色**（warning）：资料缺失、未完成或需注意的局部提示。
- **暗红**（danger）：遗忘等破坏性操作。状态同时提供文字说明。

## Typography

**Display Font:** 无独立展示字体角色。
**Body Font:** 系统无衬线栈；中文优先由 PingFang SC / Microsoft YaHei 承接。
**Label/Mono Font:** 标签沿用系统字体；代码使用 ui-monospace、SFMono-Regular、Consolas、monospace。

同一字族通过字号、字重与留白组织阅读，不以装饰性字体区分平台来源。界面根字号与阅读正文是两个角色：紧凑控件不能决定长文的大小。

### Hierarchy

- **Headline**：页面标题；手机缩为（24px）。
- **Title**：桌面会话标题；中等窄屏缩为（17px），手机正文标题为（16px）。记忆正文标题使用（23px），手机为（20px）。
- **Body**：以前置正文 token 为准，最大行长（76ch）；手机与平板在（800px）以下使用 body-mobile。
- **Label**：常规说明与控件采用 label；来源和时间在现有组件中更紧凑，不作为新正文规格。
- **Code**：代码块独立等宽排版、浅色底、内部滚动；行内代码随正文缩放。

**The 阅读优先 Rule.** 正文使用独立于紧凑界面标签的字号与行距；新增页面继承阅读正文角色，不从元数据字号推导正文。

## Layout

桌面外壳使用固定侧栏（218px），主区占满剩余宽度；顶部栏（60px）固定于布局中，正文区域在视口高度内独立滚动。页面常规左右边距（32px）。会话和记忆的目录列（310px）与弹性阅读区并排；大屏（1500px 起）目录扩为（340px），正文横向内边距扩为（40px）。正文遵守最大行长，剩余空间保留为空白。

中等屏幕（1100px 以下）侧栏缩为（185px），目录列缩为（245px），页面边距（22px）。在（800px 以下）主导航改为抽屉，关闭时不可见且 inert；页面边距趋向（18px），顶部栏缩为（54px），目录列仍可与阅读区并排。

手机（540px 以下）的会话采用列表／正文切换：选中会话后隐藏列表，正文独占阅读空间，顶部有“返回会话列表”。返回列表或搜索时显示列表并隐藏正文。会话外框左右边距为（12px），正文内边距为（20px）。记忆页在此断点采用上方有限高度列表、下方正文的纵向结构，不把会话切换行为误当作所有分栏页面的现状。

设置页按横向分隔线组织段落，桌面使用说明列（240px）与内容列，列间距（40px）；在（800px 以下）转为单列。接入表格随窄屏减少次要列，保留来源及状态说明。

## Elevation & Depth

常驻界面是平面的：纸面、目录与背景通过相邻色差和细线表达层级。浮层、短暂成功提示和原生确认对话框才使用柔和阴影；精确阴影和动效记录在 sidecar。没有卡片悬浮或大面积装饰光晕。

**The 平面优先 Rule.** 常驻内容依靠色面和边线分区，阴影保留给弹出层、提示与确认对话框。

## Shapes

控件是小圆角矩形，复合输入和弹出菜单略柔和；阅读工作区仅上角圆润，下沿与连续档案内容衔接。对话框角更大。圆形仅用于细小状态点，来源与作者图标采用克制的小型图形。边框通常为（1px），选中行的左标线和活动标签下划线为（2px）。线条图标通常为（14–20px），保持一致笔画风格。

## Components

### Buttons

森林绿实心主按钮与暖白描边次按钮共用小圆角、紧凑间距。常规高度（40px）；阅读器工具条使用（32px）高度。悬停改变背景与边框，不位移；键盘焦点使用外置轮廓线。禁用态使用固定的浅色背景和可读文字，不通过整体透明度降低辨识度。安静按钮去除可见边框，危险操作保留暗红实心变体。

### Inputs / Fields

暖白底、浅灰绿边框，与按钮共用小圆角；常规高度（40px）。输入框、按钮和选择器共享控件高度、圆角与焦点 token。原生选择器保留键盘及移动端系统选项面板，收起态使用统一 Lucide 下拉箭头。路径迁移表单将标签放在输入框上方，桌面按钮与输入框底边对齐，1100px 以下纵向排布；展开容器为焦点环保留 4px 空间。搜索为带图标的复合外框，输入本身无边框，整个外框在 focus-within 时出现森林绿（2px）轮廓，偏移（2px）。桌面显示 Cmd/Ctrl+K 提示；窄屏隐藏快捷键标签。

### Navigation

会话、记忆、接入、设置保持固定顺序。导航行整行可点，线条图标、标签和右对齐数量同处一行；活动项以浅叶绿面和较重字重区分。桌面侧栏可收起为（72px）图标轨道，保留四个导航入口与悬停名称，状态保存在本机浏览器。手机抽屉带遮罩和关闭按钮；开启时限制焦点在导航内，背景 inert，Escape 关闭并返回菜单按钮；关闭后导航 inert。导航抽屉及遮罩高于短暂提示层。

### Directory Rows and Reading Containers

会话行通过细分隔线串联，选中态浅叶绿填充并加左侧标线。来源、更新时间、条数与归档情况围绕标题形成稳定次级层。消息正文按文档段落展开，作者与时间独立成行，段落之间以空白和细线分隔。没有聊天气泡。记忆目录继承同样的平面列表语言。

### Tabs and Filter Chips

原文／摘要采用文字标签，活动项加底部绿线，数量为小型浅色标签。记忆类别筛选采用小型安静按钮，活动态用浅绿底；允许换行，不横向挤压正文。

### Status and Confirmation

归档状态由来源实际 coverage、缺失和问题信息决定，不把“已保存在本地”扩展为“平台历史完整”。状态点配文字；错误为暖色横幅，成功为底部短暂提示。遗忘使用原生 dialog，先展示影响范围再提供危险按钮。Markdown 保持文档阅读，不执行原始 HTML；链接、代码、引用、表格使用各自清晰语义。

### Motion

动效只帮助辨认状态。控件状态采用（140ms），面板、抽屉与侧栏采用（240ms）及 cubic-bezier(0.16, 1, 0.3, 1)，加载图标采用（900ms）连续旋转。异步按钮保留原宽度，在忙碌时禁用并覆盖进度文案；搜索在读取完成前不显示空结果。侧栏桌面收放会实际让出阅读宽度，因此仅此操作允许侧栏 width 与工作区 margin-left 的短过渡；其余控件使用 transform、opacity 或展开区域的 grid-template-rows。平滑滚动用于阅读定位，prefers-reduced-motion 时关闭过渡、动画和平滑滚动。

代码中的可执行 token 位于 `web/src/style.css` 的 `:root`，共用控件位于 `web/src/ui.tsx`：SelectField、AsyncButton、Disclosure、CopyButton、LoadingIndicator。尺寸以 control-height（40px）、control-compact（32px）、icon-size（16px）为基础；间距采用 4/8/12/16/24/32px。新界面复用这些控件，不再自行拼装内联表单或默认下拉箭头。

## Do's and Don'ts

### Do:

- **Do** 保留来源、时间、归档范围与证据入口的次级层次，并使用可辨认的灰绿文字。
- **Do** 让正文沿纸面连续阅读；代码和宽表格在自身区域横向滚动。
- **Do** 让键盘焦点可见；复合搜索框的焦点环落在整个外框。
- **Do** 在手机会话正文提供返回列表，并在关闭导航时将其移出交互顺序。
- **Do** 让归档状态反映来源实际范围，使用文字解释异常与缺失。

### Don't:

- **Don't** 用展示型巨标题、指标卡片或装饰性动效改写阅读室的工作气质。
- **Don't** 将手机会话目录固定堆在正文上方，挤压已选会话的阅读空间。
- **Don't** 把来源色点当作唯一状态说明。
- **Don't** 把已有小号时间和标签作为新长文正文的字号依据。
