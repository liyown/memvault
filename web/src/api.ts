export type Attachment = {
  name: string;
  blob?: string;
  missing?: boolean;
  url?: string;
  size?: number;
};
export type Message = {
  id: string;
  role: string;
  text: string;
  time: string;
  attachments?: Attachment[];
  complete: boolean;
};
export type Session = {
  id: string;
  native_id: string;
  agent: string;
  title: string;
  created: string;
  updated: string;
  coverage: string;
  issues?: string[];
  count: number;
  missing: number;
  messages?: Message[];
};
export type Memory = {
  id: string;
  title: string;
  kind: string;
  body: string;
  session?: string;
  evidence?: { session: string; message: string }[];
  covered?: string[];
  parents: string[];
  revision: string;
  updated: string;
  author: string;
  pending?: boolean;
};
export type SourceStatus = {
  agent: string;
  state: string;
  sessions: number;
  last_scan: string;
  error?: string;
};
export type Integration = {
  agent: string;
  configured: boolean;
  state: string;
  error?: string;
};
export type Status = {
  root: string;
  version: string;
  sessions: number;
  messages: number;
  memories: number;
  sources: SourceStatus[];
  issues: string[];
  time: string;
  integrations?: Integration[];
};
export type Detail = {
  session: Session;
  summaries: Memory[];
  relations: { from: string; to: string; kind: string }[];
  offset: number;
};
export type Hit = {
  session?: string;
  memory?: string;
  message?: string;
  title: string;
  agent?: string;
  role: string;
  text: string;
  time: string;
};
export type Source = { agent: string; path: string; enabled: boolean };
export type Act = (
  fn: () => Promise<unknown>,
  success?: string,
) => Promise<void>;

export const names: Record<string, string> = {
  codex: "Codex",
  claude: "Claude Code",
  opencode: "OpenCode",
};
export const kinds: Record<string, string> = {
  background: "个人背景",
  preference: "偏好与约定",
  knowledge: "知识与结论",
  summary: "会话摘要",
};

const valid = (value?: string) => !!value && !Number.isNaN(Date.parse(value));

export const date = (value?: string) =>
  valid(value)
    ? new Intl.DateTimeFormat("zh-CN", {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      }).format(new Date(value!))
    : "时间未提供";

const DAY = 86_400_000;

/** Compact relative time for directory rows: 14:02 / 昨天 / 10月3日 / 2025/10/3. */
export const age = (value?: string) => {
  if (!valid(value)) return "";
  const d = new Date(value!);
  const now = new Date();
  const startOfToday = new Date(now).setHours(0, 0, 0, 0);
  if (d.getTime() >= startOfToday)
    return new Intl.DateTimeFormat("zh-CN", {
      hour: "2-digit",
      minute: "2-digit",
    }).format(d);
  if (d.getTime() >= startOfToday - DAY) return "昨天";
  return new Intl.DateTimeFormat(
    "zh-CN",
    d.getFullYear() === now.getFullYear()
      ? { month: "numeric", day: "numeric" }
      : { year: "numeric", month: "numeric", day: "numeric" },
  ).format(d);
};

const generated =
  /^(claude|codex|opencode)?[\s·:_-]*[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}$/i;

/** Turns the first raw prompt line into a readable title; empty when it carries nothing. */
const cleanTitle = (raw: string) =>
  raw
    .replace(/\[([^\]]*)\]\([^)]*\)/g, "$1") // markdown links keep their text
    .replace(/<\/?[a-z_][\w-]*>/gi, " ") // injected wrapper tags
    .replace(/^\s*(工具结果|tool result)[:：]?\s*/i, "")
    .replace(/[`*#>{}[\]]/g, "")
    .replace(/\s+/g, " ")
    .trim();

/** Sessions without a real title carry the native id or a wrapper; show something readable. */
export const sessionTitle = (
  s: Pick<Session, "title" | "agent" | "created">,
) => {
  const title = generated.test(s.title?.trim() ?? "")
    ? ""
    : cleanTitle(s.title ?? "");
  if (title.length >= 2) return title;
  const when = valid(s.created) ? ` · ${date(s.created)}` : "";
  return `未命名的 ${names[s.agent] ?? s.agent} 会话${when}`;
};

export const sourceState: Record<string, { label: string; tone: Tone }> = {
  ready: { label: "正在采集", tone: "good" },
  paused: { label: "已暂停", tone: "neutral" },
  unavailable: { label: "当前不可用", tone: "warning" },
  attention: { label: "需要处理", tone: "warning" },
  importing: { label: "正在导入", tone: "neutral" },
};
export const integrationState: Record<string, { label: string; tone: Tone }> = {
  not_configured: { label: "尚未配置", tone: "warning" },
  configured_not_observed: { label: "已配置，等待首次活动", tone: "neutral" },
  waiting_hook_review: { label: "等待平台信任 hook", tone: "warning" },
  hook_observed: { label: "已生效", tone: "good" },
  client_not_installed: { label: "未安装客户端", tone: "neutral" },
};
export type Tone = "good" | "warning" | "danger" | "neutral";

export async function api<T>(
  path: string,
  body?: unknown,
  method?: string,
): Promise<T> {
  const r = await fetch("/api/v1/" + path, {
    method: method ?? (body === undefined ? "GET" : "POST"),
    headers: { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!r.ok) {
    let text = "请求未完成";
    try {
      text = (await r.json()).error ?? text;
    } catch {
      text = await r.text().catch(() => text);
    }
    throw Error(text);
  }
  return r.json();
}
