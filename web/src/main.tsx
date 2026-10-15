import React, {
  useCallback,
  useDeferredValue,
  useEffect,
  useRef,
  useState,
} from "react";
import { createRoot } from "react-dom/client";
import Markdown from "react-markdown";
import {
  Archive,
  BookOpen,
  Search,
  Settings,
  Plug,
  ArrowUpRight,
  ArrowLeft,
  ChevronDown,
  ChevronRight,
  Download,
  Upload,
  RefreshCw,
  Plus,
  Check,
  Folder,
  Link,
  History,
  Trash2,
  FileText,
  AlertCircle,
  Copy,
  X,
  Menu,
  ArrowRight,
  ShieldCheck,
  Clock,
  PanelLeftClose,
  PanelLeftOpen,
  LoaderCircle,
} from "lucide-react";
import {
  AsyncButton,
  CopyButton,
  Disclosure,
  LoadingIndicator,
  SelectField,
} from "./ui";
import "./style.css";

type Attachment = {
  name: string;
  blob?: string;
  missing?: boolean;
  url?: string;
  size?: number;
};
type Message = {
  id: string;
  role: string;
  text: string;
  time: string;
  attachments?: Attachment[];
  complete: boolean;
};
type Session = {
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
type Memory = {
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
type SourceStatus = {
  agent: string;
  state: string;
  sessions: number;
  last_scan: string;
  error?: string;
};
type Status = {
  root: string;
  version: string;
  sessions: number;
  messages: number;
  memories: number;
  sources: SourceStatus[];
  issues: string[];
  time: string;
  integrations?: {
    agent: string;
    configured: boolean;
    state: string;
    error?: string;
  }[];
};
type Detail = {
  session: Session;
  summaries: Memory[];
  relations: { from: string; to: string; kind: string }[];
  offset: number;
};
type Hit = {
  session?: string;
  memory?: string;
  message?: string;
  title: string;
  agent?: string;
  role: string;
  text: string;
  time: string;
};
type Source = { agent: string; path: string; enabled: boolean };
const names: Record<string, string> = {
  codex: "Codex",
  claude: "Claude Code",
  opencode: "OpenCode",
};
const kinds: Record<string, string> = {
  background: "个人背景",
  preference: "偏好与约定",
  knowledge: "知识与结论",
  summary: "会话摘要",
};
const date = (value?: string) =>
  value && !Number.isNaN(Date.parse(value))
    ? new Intl.DateTimeFormat("zh-CN", {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      }).format(new Date(value))
    : "时间未提供";
const age = (value?: string) =>
  value && !Number.isNaN(Date.parse(value))
    ? new Intl.DateTimeFormat("zh-CN", {
        month: "numeric",
        day: "numeric",
      }).format(new Date(value))
    : "";
async function api<T>(
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
function Prose({ text }: { text: string }) {
  return (
    <div className="prose">
      <Markdown
        components={{
          a: ({ children, ...props }) => (
            <a {...props} target="_blank" rel="noreferrer">
              {children}
            </a>
          ),
        }}
      >
        {text}
      </Markdown>
    </div>
  );
}
function Empty({
  title,
  body,
  icon = "book",
}: {
  title: string;
  body: string;
  icon?: string;
}) {
  return (
    <div className="empty">
      {icon === "book" ? (
        <BookOpen size={34} strokeWidth={1.3} />
      ) : (
        <Search size={34} strokeWidth={1.3} />
      )}
      <h3>{title}</h3>
      <p>{body}</p>
    </div>
  );
}
function App() {
  const [narrow, setNarrow] = useState(
    () => window.matchMedia("(max-width:800px)").matches,
  );
  useEffect(() => {
    const media = window.matchMedia("(max-width:800px)");
    const update = () => setNarrow(media.matches);
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);
  const [auth, setAuth] = useState<"loading" | "ready" | "required">("loading");
  const [page, setPage] = useState("sessions");
  const [status, setStatus] = useState<Status>();
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [tick, setTick] = useState(0);
  const [mobile, setMobile] = useState(false);
  const mobileWasOpen = useRef(false);
  const [collapsed, setCollapsed] = useState(() => {
    try {
      return localStorage.getItem("memvault.sidebar.v1") === "collapsed";
    } catch {
      return false;
    }
  });
  const [refreshing, setRefreshing] = useState(false);
  const [importing, setImporting] = useState(false);
  const toggleSidebar = () => {
    if (narrow) setMobile((open) => !open);
    else
      setCollapsed((value) => {
        const next = !value;
        try {
          localStorage.setItem(
            "memvault.sidebar.v1",
            next ? "collapsed" : "expanded",
          );
        } catch {
          /* Storage can be unavailable in a private window. */
        }
        return next;
      });
  };
  const [selected, setSelected] = useState(
    new URLSearchParams(location.search).get("session") ?? "",
  );
  const [targetMessage, setTargetMessage] = useState("");
  const upload = useRef<HTMLInputElement>(null);
  useEffect(() => {
    const hash = new URLSearchParams(location.hash.slice(1));
    const token = hash.get("token");
    (async () => {
      try {
        if (token) {
          const r = await fetch("/api/v1/auth", {
            method: "POST",
            headers: { Authorization: "Bearer " + token },
          });
          if (!r.ok) throw Error();
          history.replaceState({}, "", location.pathname + location.search);
        }
        await api("status");
        setAuth("ready");
      } catch {
        setAuth("required");
      }
    })();
  }, []);
  const refresh = useCallback(() => setTick((t) => t + 1), []);
  useEffect(() => {
    if (auth !== "ready") return;
    let cancelled = false;
    setRefreshing(true);
    api<Status>("status")
      .then((s) => {
        if (!cancelled) setStatus(s);
      })
      .catch((e) => {
        if (!cancelled) setError(e.message);
      })
      .finally(() => {
        if (!cancelled) setRefreshing(false);
      });
    return () => {
      cancelled = true;
    };
  }, [auth, tick]);
  useEffect(() => {
    if (auth !== "ready") return;
    const es = new EventSource("/api/v1/events");
    let timer: ReturnType<typeof setTimeout> | undefined;
    es.addEventListener("change", () => {
      if (timer) clearTimeout(timer);
      timer = setTimeout(refresh, 600);
    });
    return () => {
      es.close();
      if (timer) clearTimeout(timer);
    };
  }, [auth, refresh]);
  useEffect(() => {
    if (!notice) return;
    const t = setTimeout(() => setNotice(""), 4000);
    return () => clearTimeout(t);
  }, [notice]);
  useEffect(() => {
    if (!mobile || !narrow) {
      if (mobileWasOpen.current) {
        mobileWasOpen.current = false;
        document.querySelector<HTMLButtonElement>(".sidebar-toggle")?.focus();
      }
      return;
    }
    mobileWasOpen.current = true;
    const nav = document.getElementById("main-navigation");
    nav?.querySelector<HTMLButtonElement>("nav button")?.focus();
    const escape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setMobile(false);
        document.querySelector<HTMLButtonElement>(".sidebar-toggle")?.focus();
      }
      if (e.key === "Tab") {
        const controls = nav?.querySelectorAll<HTMLElement>(
          "a[href], button:not([disabled])",
        );
        const first = controls?.[0],
          last = controls?.[controls.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last?.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first?.focus();
        }
      }
    };
    window.addEventListener("keydown", escape);
    return () => window.removeEventListener("keydown", escape);
  }, [mobile, narrow]);
  const goSession = (id: string, message = "") => {
    setSelected(id);
    setTargetMessage(message);
    setPage("sessions");
    history.replaceState({}, "", "?session=" + encodeURIComponent(id));
  };
  const act = async (fn: () => Promise<unknown>, success = "已完成") => {
    try {
      await fn();
      setNotice(success);
      refresh();
    } catch (e) {
      setError((e as Error).message);
    }
  };
  const importFile = async (file?: File) => {
    if (!file) return;
    if (importing) return;
    setImporting(true);
    try {
      const r = await fetch("/api/v1/import", {
        method: "POST",
        headers: { "Content-Type": "application/zip" },
        body: file,
      });
      if (!r.ok) throw Error((await r.json()).error);
      setNotice("档案已校验并导入");
      refresh();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setImporting(false);
    }
  };
  const showMemory = (id: string) => {
    setPage("memories");
    setMemoryTarget(id);
  };
  const [memoryTarget, setMemoryTarget] = useState("");
  if (auth === "loading")
    return (
      <div className="gate">
        <BookOpen size={32} />
        <LoadingIndicator label="正在打开你的记忆库…" />
      </div>
    );
  if (auth === "required")
    return (
      <div className="gate">
        <BookOpen size={38} />
        <h1>你的记忆，只在本机打开</h1>
        <p>在终端运行下面的命令，安全打开这个页面。</p>
        <code>memvault open</code>
        <button onClick={() => location.reload()}>
          <RefreshCw size={16} />
          重新连接
        </button>
      </div>
    );
  const nav = [
    { id: "sessions", label: "会话", icon: Archive, count: status?.sessions },
    { id: "memories", label: "记忆", icon: BookOpen, count: status?.memories },
    { id: "sources", label: "接入", icon: Plug },
    { id: "settings", label: "设置", icon: Settings },
  ];
  const problem = (status?.issues?.length ?? 0) > 0;
  const sourceErrors =
    status?.sources?.filter((s) => s.state === "attention").length ?? 0;
  return (
    <div
      className={`app ${mobile && narrow ? "nav-open" : ""} ${collapsed ? "nav-collapsed" : ""}`}
    >
      <button
        className="nav-scrim"
        aria-label="关闭导航遮罩"
        tabIndex={-1}
        onClick={() => {
          setMobile(false);
          document.querySelector<HTMLButtonElement>(".sidebar-toggle")?.focus();
        }}
      />
      <aside className="sidebar" id="main-navigation" inert={narrow && !mobile}>
        <button
          className="icon-button drawer-close"
          aria-label="关闭导航"
          onClick={() => {
            setMobile(false);
            document
              .querySelector<HTMLButtonElement>(".sidebar-toggle")
              ?.focus();
          }}
        >
          <X size={18} />
        </button>
        <a className="brand" href="/" aria-label="Memvault 首页">
          <span className="brand-mark">
            <BookOpen size={23} />
          </span>
          <span>
            Memvault<span className="brand-sub">你的会话与记忆</span>
          </span>
        </a>
        <nav aria-label="主要导航">
          {nav.map(({ id, label, icon: Icon, count }) => (
            <button
              className={"nav-item " + (page === id ? "active" : "")}
              key={id}
              aria-label={label}
              aria-current={page === id ? "page" : undefined}
              title={collapsed && !narrow ? label : undefined}
              onClick={() => {
                setPage(id);
                setMobile(false);
              }}
            >
              <Icon size={18} strokeWidth={1.7} />
              <span>{label}</span>
              {count !== undefined && <small>{count.toLocaleString()}</small>}
            </button>
          ))}
        </nav>
        <div className="sidebar-bottom">
          <div className="local-note">
            <span className="live-dot" />
            本地运行
          </div>
          <p>
            会话归你所有。
            <br />
            记忆随你同行。
          </p>
          <button
            className="vault-location"
            title={status?.root}
            onClick={() => setPage("settings")}
          >
            <Folder size={15} />
            <span>
              {status?.root.split("/").filter(Boolean).pop() ?? "记忆位置"}
            </span>
            <ChevronRight size={14} />
          </button>
          <small className="version">{status?.version ?? "preview"}</small>
        </div>
      </aside>
      <div className="workspace" inert={narrow && mobile}>
        <header className="topbar">
          <button
            className="icon-button sidebar-toggle"
            aria-label={
              (narrow ? mobile : !collapsed) ? "收起导航" : "展开导航"
            }
            title={(narrow ? mobile : !collapsed) ? "收起导航" : "展开导航"}
            aria-expanded={narrow ? mobile : !collapsed}
            aria-controls="main-navigation"
            onClick={toggleSidebar}
          >
            {narrow ? (
              <Menu size={20} />
            ) : collapsed ? (
              <PanelLeftOpen size={18} />
            ) : (
              <PanelLeftClose size={18} />
            )}
          </button>
          <span className="breadcrumb">
            个人记忆库<span>/</span>
            <strong>{nav.find((n) => n.id === page)?.label}</strong>
          </span>
          <div className="protection">
            <ShieldCheck size={15} />
            <span>
              {problem || sourceErrors
                ? "部分资料需要处理"
                : "原始资料保存在本地"}
            </span>
          </div>
          <button
            className="icon-button"
            aria-label="刷新状态"
            onClick={refresh}
            disabled={refreshing}
            aria-busy={refreshing}
          >
            <span className={refreshing ? "spinner" : "icon-slot"}>
              <RefreshCw size={16} />
            </span>
          </button>
        </header>
        {error && (
          <div className="error-banner" role="alert">
            <AlertCircle size={18} />
            <span>{error}</span>
            <button aria-label="关闭错误提示" onClick={() => setError("")}>
              <X size={16} />
            </button>
          </div>
        )}
        {importing && (
          <div className="operation-status">
            <LoadingIndicator label="正在校验并导入档案…" />
          </div>
        )}
        {notice && (
          <div className="toast" role="status">
            <Check size={16} />
            {notice}
          </div>
        )}
        <input
          className="file-input"
          ref={upload}
          type="file"
          accept=".zip"
          aria-label="选择记忆档案"
          onChange={(e) => {
            void importFile(e.target.files?.[0]);
            e.target.value = "";
          }}
        />
        <main>
          {page === "sessions" ? (
            <Sessions
              tick={tick}
              selected={selected}
              targetMessage={targetMessage}
              choose={goSession}
              setError={setError}
              notice={setNotice}
              importAction={() => upload.current?.click()}
              showMemory={showMemory}
            />
          ) : page === "memories" ? (
            <Memories
              tick={tick}
              target={memoryTarget}
              goSession={goSession}
              act={act}
            />
          ) : page === "sources" ? (
            <Sources status={status} act={act} />
          ) : (
            <SettingsPage
              status={status}
              act={act}
              importAction={() => upload.current?.click()}
            />
          )}
        </main>
        <footer className="app-footer">
          <span>
            <span className="live-dot" />
            {status?.messages.toLocaleString() ?? "—"} 条消息已进入本地档案
          </span>
          <span>同步与备份由你掌握</span>
        </footer>
      </div>
    </div>
  );
}
function Sessions({
  tick,
  selected,
  targetMessage,
  choose,
  setError,
  notice,
  importAction,
  showMemory,
}: {
  tick: number;
  selected: string;
  targetMessage: string;
  choose: (id: string, message?: string) => void;
  setError: (s: string) => void;
  notice: (s: string) => void;
  importAction: () => void;
  showMemory: (id: string) => void;
}) {
  const [items, setItems] = useState<Session[]>([]);
  const [total, setTotal] = useState(0);
  const [mobileList, setMobileList] = useState(false);
  const selectSession = (id: string, message = "") => {
    setMobileList(false);
    choose(id, message);
  };
  const [query, setQuery] = useState("");
  const deferred = useDeferredValue(query);
  useEffect(() => {
    if (deferred) setMobileList(true);
  }, [deferred]);
  const [agent, setAgent] = useState("");
  const [role, setRole] = useState("");
  const [sort, setSort] = useState("recent");
  const [hits, setHits] = useState<Hit[]>([]);
  const [loading, setLoading] = useState(true);
  const [offset, setOffset] = useState(0);
  const searchRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    const listener = (ev: KeyboardEvent) => {
      if ((ev.metaKey || ev.ctrlKey) && ev.key === "k") {
        ev.preventDefault();
        searchRef.current?.focus();
      }
    };
    window.addEventListener("keydown", listener);
    return () => window.removeEventListener("keydown", listener);
  }, []);
  useEffect(() => {
    let ignore = false;
    setLoading(true);
    const p = deferred
      ? api<{ items: Hit[] }>(
          "search?q=" +
            encodeURIComponent(deferred) +
            "&agent=" +
            agent +
            "&role=" +
            role +
            "&sort=" +
            sort +
            "&limit=40",
        )
      : api<{ items: Session[]; total: number }>(
          "sessions?agent=" + agent + "&offset=" + offset + "&limit=50",
        );
    p.then((value) => {
      if (ignore) return;
      if ("total" in value) {
        setItems(value.items ?? []);
        setTotal(value.total);
        if (!selected && value.items?.[0]) choose(value.items[0].id);
      } else setHits(value.items ?? []);
    })
      .catch((e) => {
        if (!ignore) setError(e.message);
      })
      .finally(() => {
        if (!ignore) setLoading(false);
      });
    return () => {
      ignore = true;
    };
  }, [tick, deferred, agent, role, offset, sort]);
  return (
    <section className="sessions-page">
      <div className="page-heading">
        <div>
          <h1>会话</h1>
          <p>每一次讨论，都有迹可循。</p>
        </div>
        <div className="actions">
          <button onClick={importAction}>
            <Upload size={16} />
            导入档案
          </button>
          <a className="button" href="/api/v1/export">
            <Download size={16} />
            导出记忆库
          </a>
        </div>
      </div>
      <div className="search-toolbar">
        <label className="search-box">
          <Search size={18} />
          <input
            ref={searchRef}
            placeholder="搜索会话、原文与记忆…"
            aria-label="搜索会话、原文与记忆"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          {query ? (
            <button aria-label="清空搜索" onClick={() => setQuery("")}>
              <X size={15} />
            </button>
          ) : (
            <kbd>⌘ K</kbd>
          )}
        </label>
        <SelectField
          aria-label="按来源过滤"
          value={agent}
          onChange={(e) => {
            setAgent(e.target.value);
            setOffset(0);
          }}
        >
          <option value="">全部来源</option>
          {Object.entries(names).map(([id, n]) => (
            <option value={id} key={id}>
              {n}
            </option>
          ))}
        </SelectField>
        {deferred && (
          <>
            <SelectField
              aria-label="搜索结果排序"
              value={sort}
              onChange={(e) => setSort(e.target.value)}
            >
              <option value="recent">最近讨论</option>
              <option value="relevance">相关度</option>
            </SelectField>
            <SelectField
              aria-label="按角色过滤"
              value={role}
              onChange={(e) => setRole(e.target.value)}
            >
              <option value="">所有角色</option>
              <option value="user">用户</option>
              <option value="assistant">助手</option>
              <option value="memory">记忆</option>
            </SelectField>
          </>
        )}
      </div>
      <div
        className={
          "reading-workspace " +
          (selected ? "has-selection" : "") +
          (mobileList ? " show-mobile-list" : "")
        }
      >
        <div className="session-list">
          <div className="list-heading">
            <span>{deferred ? "搜索结果" : "最近讨论"}</span>
            <small aria-live="polite">
              {loading ? (
                <LoadingIndicator label="正在读取…" />
              ) : deferred ? (
                `${hits.length} 条匹配`
              ) : (
                `${total.toLocaleString()} 个会话`
              )}
            </small>
          </div>
          {loading && (deferred ? hits.length === 0 : items.length === 0) ? (
            <div
              className="skeleton-list"
              role="status"
              aria-label="正在读取会话"
            >
              <span className="sr-only">正在读取会话…</span>
              {[1, 2, 3, 4].map((n) => (
                <div key={n} />
              ))}
            </div>
          ) : deferred ? (
            hits.length ? (
              hits.map((h, i) => (
                <button
                  key={`${h.session}-${h.message}-${i}`}
                  className={
                    "session-row search-result " +
                    (selected === h.session ? "selected" : "")
                  }
                  onClick={() =>
                    h.memory
                      ? showMemory(h.memory)
                      : h.session
                        ? selectSession(h.session, h.message)
                        : undefined
                  }
                >
                  <div className="row-meta">
                    <span>
                      {h.memory ? "记忆" : (names[h.agent ?? ""] ?? h.agent)}
                    </span>
                    <time>{age(h.time)}</time>
                  </div>
                  <h3>{h.title}</h3>
                  <p>{h.text}</p>
                  <span className="source-link">
                    {h.memory ? "长期记忆" : "查看原文"}
                    <ArrowUpRight size={13} />
                  </span>
                </button>
              ))
            ) : (
              <Empty
                title="没有找到匹配"
                body="试试更短的关键词，或取消来源和角色过滤。"
                icon="search"
              />
            )
          ) : items.length ? (
            items.map((s) => (
              <button
                key={s.id}
                className={
                  "session-row " + (selected === s.id ? "selected" : "")
                }
                onClick={() => selectSession(s.id)}
              >
                <div className="row-meta">
                  <span className={"agent-label " + s.agent}>
                    <span />
                    {names[s.agent] ?? s.agent}
                  </span>
                  <time>{age(s.updated)}</time>
                </div>
                <h3>{s.title}</h3>
                <div className="row-bottom">
                  <span>{s.count} 条消息</span>
                  {s.missing > 0 ? (
                    <span className="muted-warning">附件待补齐</span>
                  ) : (
                    <span>原文已归档</span>
                  )}
                </div>
              </button>
            ))
          ) : (
            <Empty
              title="从第一次讨论开始"
              body="后台会自动发现已接入 agent 的会话。也可以导入已有的 Memvault 档案。"
            />
          )}
          {!deferred && total > 50 && (
            <div className="pagination">
              <button
                disabled={offset === 0}
                onClick={() => setOffset(Math.max(0, offset - 50))}
              >
                上一页
              </button>
              <span>
                {Math.floor(offset / 50) + 1} / {Math.ceil(total / 50)}
              </span>
              <button
                disabled={offset + 50 >= total}
                onClick={() => setOffset(offset + 50)}
              >
                下一页
              </button>
            </div>
          )}
        </div>
        <div className="reading-pane">
          <button className="mobile-back" onClick={() => setMobileList(true)}>
            <ArrowLeft size={16} />
            返回会话列表
          </button>
          {selected ? (
            <SessionReader
              id={selected}
              targetMessage={targetMessage}
              tick={tick}
              choose={choose}
              setError={setError}
              notice={notice}
            />
          ) : (
            <Empty
              title="留住讨论，也留住来龙去脉"
              body="选择一段会话，阅读原文、回顾结论，或交给另一个 agent 继续。"
            />
          )}
        </div>
      </div>
    </section>
  );
}
function SessionReader({
  id,
  targetMessage,
  tick,
  choose,
  setError,
  notice,
}: {
  id: string;
  targetMessage: string;
  tick: number;
  choose: (id: string, message?: string) => void;
  setError: (s: string) => void;
  notice: (s: string) => void;
}) {
  const [detail, setDetail] = useState<Detail>();
  const [tab, setTab] = useState("transcript");
  const [offset, setOffset] = useState(0);
  const [resume, setResume] = useState(false);
  const [busy, setBusy] = useState(false);
  const [forget, setForget] = useState(false);
  const [pack, setPack] = useState("");
  const scroller = useRef<HTMLDivElement>(null);
  const resumeMenu = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!resume) return;
    const root = resumeMenu.current;
    root?.querySelector<HTMLButtonElement>('[role="menuitem"]')?.focus();
    const dismiss = (event: PointerEvent) => {
      if (!root?.contains(event.target as Node)) setResume(false);
    };
    const keyboard = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        setResume(false);
        root
          ?.querySelector<HTMLButtonElement>('[aria-haspopup="menu"]')
          ?.focus();
      } else if (event.key === "Tab") setResume(false);
      else if (["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) {
        const choices = Array.from(
          root?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? [],
        );
        if (!choices.length) return;
        event.preventDefault();
        const current = choices.indexOf(
          document.activeElement as HTMLButtonElement,
        );
        const next =
          event.key === "Home"
            ? 0
            : event.key === "End"
              ? choices.length - 1
              : (current +
                  (event.key === "ArrowUp" ? -1 : 1) +
                  choices.length) %
                choices.length;
        choices[next]?.focus();
      }
    };
    document.addEventListener("pointerdown", dismiss);
    root?.addEventListener("keydown", keyboard);
    return () => {
      document.removeEventListener("pointerdown", dismiss);
      root?.removeEventListener("keydown", keyboard);
    };
  }, [resume]);
  useEffect(() => {
    setOffset(0);
    setDetail(undefined);
    setPack("");
    setTab("transcript");
  }, [id, targetMessage]);
  useEffect(() => {
    let ignore = false;
    let path = `sessions/${id}?offset=${offset}&limit=100`;
    if (targetMessage && offset === 0)
      path += "&message=" + encodeURIComponent(targetMessage);
    api<Detail>(path)
      .then((d) => {
        if (!ignore) setDetail(d);
      })
      .catch((e) => {
        if (!ignore) setError(e.message);
      });
    return () => {
      ignore = true;
    };
  }, [id, targetMessage, offset, tick]);
  useEffect(() => {
    if (!targetMessage) return;
    const t = setTimeout(
      () =>
        document
          .getElementById("msg-" + targetMessage)
          ?.scrollIntoView({ block: "center", behavior: "smooth" }),
      100,
    );
    return () => clearTimeout(t);
  }, [detail, targetMessage]);
  const start = async (agent: string) => {
    setBusy(true);
    setResume(false);
    try {
      const value = await api<{
        launched: boolean;
        context: string;
        reason?: string;
      }>("resume", { session: id, agent, launch: true });
      if (value.launched) notice(`已在终端打开 ${names[agent]}`);
      else {
        setPack(value.context);
        notice(value.reason ?? "接续材料已生成");
      }
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
      setResume(false);
    }
  };
  if (!detail)
    return (
      <div className="reader-loading" role="status" aria-label="正在读取原文">
        <span className="sr-only">正在读取原文…</span>
        <div />
        <div />
        <div />
      </div>
    );
  const s = detail.session;
  const summaries = detail.summaries ?? [];
  const messages = s.messages ?? [];
  return (
    <>
      <div className="reader-header">
        <div className="reader-source">
          <span className={"agent-label " + s.agent}>
            <span />
            {names[s.agent]}
          </span>
          <span>·</span>
          <time>{date(s.created)}</time>
          <span className="saved-label">
            <Check size={13} />
            本地档案
          </span>
        </div>
        <h2>{s.title}</h2>
        <div className="reader-actions">
          <div className="resume-wrapper" ref={resumeMenu}>
            <button
              className="primary"
              aria-expanded={resume}
              aria-haspopup="menu"
              onKeyDown={(e) => {
                if (e.key === "ArrowDown") {
                  e.preventDefault();
                  setResume(true);
                }
              }}
              onClick={() => setResume(!resume)}
              disabled={busy}
            >
              {busy ? (
                <span className="spinner">
                  <LoaderCircle size={15} />
                </span>
              ) : (
                <ArrowUpRight size={15} />
              )}
              {busy ? "正在打开…" : "继续会话"}
              <ChevronDown size={14} />
            </button>
            {resume && (
              <div className="popover" role="menu" aria-label="选择接续 agent">
                {Object.entries(names).map(([key, name]) => (
                  <button
                    key={key}
                    role="menuitem"
                    onClick={() => void start(key)}
                  >
                    用 {name} 继续
                    <ArrowUpRight size={14} />
                  </button>
                ))}
              </div>
            )}
          </div>
          <a
            className="button quiet"
            href={`/api/v1/sessions/${id}?format=markdown`}
          >
            <Download size={15} />
            导出原文
          </a>
          <button
            className="icon-button"
            aria-label="遗忘这段会话"
            title="遗忘"
            onClick={() => setForget(true)}
          >
            <Trash2 size={16} />
          </button>
        </div>
      </div>
      <div className="reader-tabs">
        <button
          className={tab === "transcript" ? "active" : ""}
          onClick={() => setTab("transcript")}
        >
          <FileText size={15} />
          对话<span>{s.count}</span>
        </button>
        <button
          className={tab === "summary" ? "active" : ""}
          onClick={() => setTab("summary")}
        >
          <BookOpen size={15} />
          摘要
        </button>
        <div className="reading-assurance">
          <ShieldCheck size={13} />
          <span>独立保留，可追溯</span>
        </div>
      </div>
      <div className="reader-scroll" ref={scroller}>
        {pack && (
          <div className="handoff-inline">
            <h3>接续材料已准备好</h3>
            <p>目标 agent 当前不可用。你可以复制这些材料，交给其他助手继续。</p>
            <button
              onClick={() =>
                void navigator.clipboard
                  .writeText(pack)
                  .then(() => notice("接续材料已复制"))
              }
            >
              <Copy size={15} />
              复制接续材料
            </button>
            <button className="quiet" onClick={() => setPack("")}>
              收起
            </button>
          </div>
        )}
        {tab === "summary" ? (
          <div className="summary-content">
            {summaries.length > 1 && (
              <div className="inline-alert">
                存在并发摘要版本，agent 会在后续讨论中合并。
              </div>
            )}
            {summaries.length ? (
              summaries.map((m) => (
                <section key={m.revision}>
                  <h3>{m.title}</h3>
                  <Prose text={m.body} />
                  <p className="reading-footnote">
                    更新于 {date(m.updated)} · {m.covered?.length ?? 0}{" "}
                    条消息已整理
                  </p>
                </section>
              ))
            ) : (
              <Empty
                title="原文已留存，摘要待整理"
                body="当前 agent 会在正常讨论中整理背景、结论与下一步。未整理的内容始终可以从对话原文读取。"
              />
            )}
          </div>
        ) : (
          <div className="transcript">
            {detail.offset > 0 && (
              <button
                className="load-messages"
                onClick={() => setOffset(Math.max(0, detail.offset - 100))}
              >
                读取更早的 100 条消息
              </button>
            )}
            {messages.map((m) => (
              <article
                id={"msg-" + m.id}
                className={
                  "message " +
                  m.role +
                  (targetMessage === m.id ? " referenced" : "")
                }
                key={m.id}
              >
                <div className="message-author">
                  <span className={"author-avatar " + m.role}>
                    {m.role === "user" ? (
                      <span>你</span>
                    ) : m.role === "tool" ? (
                      <Plug size={13} />
                    ) : (
                      <span>{(names[s.agent] ?? "A")[0]}</span>
                    )}
                  </span>
                  <strong>
                    {m.role === "user"
                      ? "你"
                      : m.role === "tool"
                        ? "工具"
                        : names[s.agent]}
                  </strong>
                  <time>{date(m.time)}</time>
                  {!m.complete && <span className="draft-label">进行中</span>}
                  <button
                    className="copy-message icon-button"
                    aria-label="复制这条消息"
                    onClick={() =>
                      void navigator.clipboard
                        .writeText(m.text)
                        .then(() => notice("消息已复制"))
                    }
                  >
                    <Copy size={13} />
                  </button>
                </div>
                <Prose text={m.text} />
                {m.attachments?.map((a, i) => (
                  <div className="attachment" key={i}>
                    <FileText size={17} />
                    <span>{a.name}</span>
                    {a.missing ? (
                      <small>尚未取得副本</small>
                    ) : (
                      <a
                        href={`/api/v1/objects/${a.blob}?name=${encodeURIComponent(a.name)}`}
                      >
                        <Download size={14} />
                        下载本地副本
                      </a>
                    )}
                  </div>
                ))}
              </article>
            ))}
            {detail.offset + messages.length < s.count && (
              <div className="more-messages">
                <button onClick={() => setOffset(detail.offset + 100)}>
                  读取接下来的 100 条<ArrowRight size={15} />
                </button>
                <button
                  className="quiet"
                  onClick={() => setOffset(Math.max(0, s.count - 100))}
                >
                  跳到最新
                </button>
              </div>
            )}
          </div>
        )}
        <div className="archive-note">
          <ShieldCheck size={15} />
          <p>
            {s.coverage}
            {(s.issues?.length ?? 0) > 0 && (
              <span className="muted-warning"> · {s.issues?.join("；")}</span>
            )}
          </p>
        </div>
        {(detail.relations?.length ?? 0) > 0 && (
          <div className="relations">
            <h4>关联会话</h4>
            {detail.relations.map((r, i) => (
              <button
                key={i}
                onClick={() => choose(r.from === id ? r.to : r.from)}
              >
                <Link size={14} />
                {r.kind === "continuation"
                  ? "接续"
                  : r.kind === "branch"
                    ? "分支"
                    : "引用"}{" "}
                · {(r.from === id ? r.to : r.from).slice(0, 12)}
                <ArrowUpRight size={13} />
              </button>
            ))}
          </div>
        )}
      </div>
      {forget && (
        <ForgetDialog
          kind="session"
          id={id}
          close={() => setForget(false)}
          done={() => {
            setForget(false);
            location.assign("/");
          }}
        />
      )}
    </>
  );
}
function Memories({
  tick,
  target,
  goSession,
  act,
}: {
  tick: number;
  target: string;
  goSession: (id: string, message?: string) => void;
  act: (fn: () => Promise<unknown>, s?: string) => Promise<void>;
}) {
  const [items, setItems] = useState<Memory[]>([]);
  const [selected, setSelected] = useState<Memory>();
  const [filter, setFilter] = useState("");
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [kind, setKind] = useState("knowledge");
  const [history, setHistory] = useState<Memory[]>([]);
  const [showHistory, setShowHistory] = useState(false);
  const [forget, setForget] = useState(false);
  useEffect(() => {
    let ignore = false;
    api<{ items: Memory[] }>("memories").then((r) => {
      if (ignore) return;
      const next = r.items ?? [];
      setItems(next);
      if (target) {
        const found = next.find((m) => m.id === target);
        if (found) setSelected(found);
      }
    });
    return () => {
      ignore = true;
    };
  }, [tick, target]);
  const choose = (m: Memory) => {
    setSelected(m);
    setEditing(false);
    setShowHistory(false);
  };
  const edit = (m?: Memory) => {
    setSelected(m);
    setTitle(m?.title ?? "");
    setBody(m?.body ?? "");
    setKind(m?.kind ?? "knowledge");
    setEditing(true);
    setShowHistory(false);
  };
  const save = () =>
    act(async () => {
      const parents = items
        .filter((m) => m.id === selected?.id)
        .map((m) => m.revision);
      const m = await api<Memory>("memories", {
        ...selected,
        id: selected?.id ?? "",
        title,
        body,
        kind,
        parents,
        author: "user",
      });
      setSelected(m);
      setEditing(false);
    }, "记忆已保存，历史版本保留");
  return (
    <section className="memory-page">
      <div className="page-heading">
        <div>
          <h1>记忆</h1>
          <p>留下有效的认识，也保留它从何而来。</p>
        </div>
        <button className="primary" onClick={() => edit()}>
          <Plus size={16} />
          添加记忆
        </button>
      </div>
      <div className="memory-filters">
        <button
          className={!filter ? "active" : ""}
          onClick={() => setFilter("")}
        >
          全部
        </button>
        {Object.entries(kinds).map(([id, label]) => (
          <button
            key={id}
            className={filter === id ? "active" : ""}
            onClick={() => setFilter(id)}
          >
            {label}
          </button>
        ))}
      </div>
      <div className="memory-workspace">
        <div className="memory-list">
          {items
            .filter((m) => !filter || m.kind === filter)
            .map((m) => (
              <button
                className={
                  "memory-row " +
                  (selected?.revision === m.revision ? "selected" : "")
                }
                key={m.revision}
                onClick={() => choose(m)}
              >
                <span className="memory-kind">{kinds[m.kind]}</span>
                <h3>{m.title}</h3>
                <p>{m.body.slice(0, 100)}</p>
                <div className="row-bottom">
                  <span>{date(m.updated)}</span>
                  {m.pending && <span>待确认</span>}
                  {items.filter((x) => x.id === m.id).length > 1 && (
                    <span className="muted-warning">并发版本</span>
                  )}
                </div>
              </button>
            ))}
          {items.length === 0 && (
            <Empty
              title="记忆会在讨论中生长"
              body="agent 会自动留下你的偏好、背景和重要结论。你也可以在这里直接添加。"
            />
          )}
        </div>
        <div className="memory-detail">
          {editing ? (
            <div className="memory-editor">
              <h2>{selected ? "编辑记忆" : "添加记忆"}</h2>
              <label>
                标题
                <input
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="这条记忆关于什么？"
                />
              </label>
              <label>
                分类
                <SelectField
                  value={kind}
                  onChange={(e) => setKind(e.target.value)}
                >
                  {Object.entries(kinds)
                    .filter(([k]) => k !== "summary" || !!selected?.session)
                    .map(([k, v]) => (
                      <option key={k} value={k}>
                        {v}
                      </option>
                    ))}
                </SelectField>
              </label>
              <label>
                内容
                <textarea
                  rows={16}
                  value={body}
                  onChange={(e) => setBody(e.target.value)}
                  placeholder="用清楚的文字记录，支持 Markdown。"
                />
              </label>
              <div className="actions">
                <AsyncButton
                  className="primary"
                  disabled={!title.trim() || !body.trim()}
                  onAction={save}
                  busyLabel="正在保存…"
                >
                  <Check size={16} />
                  保存记忆
                </AsyncButton>
                <button onClick={() => setEditing(false)}>取消</button>
              </div>
            </div>
          ) : selected ? (
            <div className="memory-document">
              <div className="memory-detail-meta">
                <span>{kinds[selected.kind]}</span>
                <span>{date(selected.updated)}</span>
              </div>
              <h2>{selected.title}</h2>
              <div className="actions">
                <button onClick={() => edit(selected)}>编辑</button>
                <button
                  onClick={() => {
                    void api<{ items: Memory[] }>(
                      "memories?id=" + selected.id + "&history=true",
                    ).then((r) => {
                      setHistory(r.items ?? []);
                      setShowHistory(!showHistory);
                    });
                  }}
                >
                  <History size={15} />
                  历史版本
                </button>
                <button
                  className="icon-button"
                  aria-label="遗忘这条记忆"
                  onClick={() => setForget(true)}
                >
                  <Trash2 size={16} />
                </button>
              </div>
              <Prose text={selected.body} />
              <div className="evidence">
                <h4>原文依据</h4>
                {selected.evidence?.length ? (
                  selected.evidence.map((s, i) => (
                    <button
                      key={i}
                      onClick={() => goSession(s.session, s.message)}
                    >
                      <FileText size={15} />
                      <span>来源会话 · {s.session.slice(0, 12)}</span>
                      <ArrowUpRight size={14} />
                    </button>
                  ))
                ) : (
                  <p>由用户直接添加，尚未关联会话原文。</p>
                )}
              </div>
              {showHistory && (
                <div className="history-panel">
                  <h3>历史版本</h3>
                  <p>每个版本保留原文，恢复会创建新版本。</p>
                  {history.map((m) => (
                    <details key={m.revision}>
                      <summary>
                        {date(m.updated)} · {m.author}
                        <ChevronDown size={14} />
                      </summary>
                      <Prose text={m.body} />
                      <button
                        onClick={() => {
                          setBody(m.body);
                          setTitle(m.title);
                          setKind(m.kind);
                          setEditing(true);
                        }}
                      >
                        以此版本继续编辑
                      </button>
                    </details>
                  ))}
                </div>
              )}
            </div>
          ) : (
            <Empty
              title="记忆可以修订，来路始终清楚"
              body="选择一条记忆，阅读内容、追溯原文或查看历史版本。"
            />
          )}
        </div>
      </div>
      {forget && selected && (
        <ForgetDialog
          kind="memory"
          id={selected.id}
          close={() => setForget(false)}
          done={() => {
            setForget(false);
            setSelected(undefined);
            location.reload();
          }}
        />
      )}
    </section>
  );
}
function Sources({
  status,
  act,
}: {
  status?: Status;
  act: (fn: () => Promise<unknown>, s?: string) => Promise<void>;
}) {
  return (
    <section className="standard-page">
      <div className="page-heading">
        <div>
          <h1>接入</h1>
          <p>不同的入口，共享同一份本地档案。</p>
        </div>
        <AsyncButton
          busyLabel="正在扫描…"
          onAction={() => act(() => api("scan", {}), "已安排重新扫描")}
        >
          <RefreshCw size={16} />
          重新扫描
        </AsyncButton>
      </div>
      <div className="section-note">
        <ShieldCheck size={20} />
        <div>
          <strong>原文先归档，摘要随后整理</strong>
          <p>来源暂时不可用不会删除已有档案。未完成的采集会自动补做。</p>
        </div>
      </div>
      <div className="source-table">
        <div className="table-heading">
          <span>来源</span>
          <span>采集状态</span>
          <span>发现的记录</span>
          <span>最近检查</span>
        </div>
        {Object.entries(names).map(([agent, name]) => {
          const s = status?.sources?.find((s) => s.agent === agent);
          return (
            <div className="source-table-row" key={agent}>
              <div>
                <span className={"source-symbol " + agent}>{name[0]}</span>
                <strong>{name}</strong>
              </div>
              <div>
                <span
                  className={
                    "state-label " + (s?.state === "ready" ? "good" : "warning")
                  }
                >
                  <span />
                  {s?.state === "ready"
                    ? "可读取"
                    : s?.state === "paused"
                      ? "已暂停"
                      : s?.state === "unavailable"
                        ? "当前不可用"
                        : s?.state === "attention"
                          ? "需要处理"
                          : s?.state === "importing"
                            ? "正在导入"
                            : "正在检测"}
                </span>
                {s?.error && <p className="source-error">{s.error}</p>}
              </div>
              <span>
                {s?.sessions?.toLocaleString() ?? "—"} <small>个记录</small>
              </span>
              <time>{date(s?.last_scan)}</time>
            </div>
          );
        })}
      </div>
      <div className="source-help">
        <h2>自动接入状态</h2>
        {status?.integrations?.map((i) => (
          <p key={i.agent}>
            <strong>{names[i.agent]}</strong> ·{" "}
            {{
              not_configured: "尚未配置",
              configured_not_observed: "已配置，等待真实活动",
              waiting_hook_review: "已配置，等待平台信任确认",
              hook_observed: "已观察到 hook 调用",
              client_not_installed: "客户端未安装，原始档案仍可读取",
            }[i.state] ?? i.state}
          </p>
        ))}
        <p>
          归档和记忆注入是两个独立环节。安装 hook 后，需要重新打开对应
          agent；Codex 首次会要求信任新增 hook。
        </p>
        <AsyncButton
          busyLabel="正在修复…"
          onAction={() =>
            act(
              () => api("repair", { action: "integrations" }),
              "接入配置已合并，已有设置保留",
            )
          }
        >
          <Plug size={16} />
          检查并修复接入配置
        </AsyncButton>
      </div>
      {status?.issues?.length ? (
        <div className="issues-panel">
          <h2>待处理事项</h2>
          {status.issues.map((x, i) => (
            <p key={i}>
              <AlertCircle size={16} />
              {x}
            </p>
          ))}
        </div>
      ) : null}
    </section>
  );
}
function SettingsPage({
  status,
  act,
  importAction,
}: {
  status?: Status;
  act: (fn: () => Promise<unknown>, s?: string) => Promise<void>;
  importAction: () => void;
}) {
  const [newRoot, setNewRoot] = useState("");
  const [sources, setSources] = useState<Source[]>([]);
  const [budget, setBudget] = useState(16000);
  const [saving, setSaving] = useState(false);
  const [settingsError, setSettingsError] = useState("");
  const [settingsLoading, setSettingsLoading] = useState(true);
  useEffect(() => {
    let ignore = false;
    api<{ sources: Source[]; context_budget: number }>("settings")
      .then((v) => {
        if (ignore) return;
        setSources(v.sources ?? []);
        setBudget(v.context_budget);
      })
      .catch((e) => {
        if (!ignore) setSettingsError(e.message);
      })
      .finally(() => {
        if (!ignore) setSettingsLoading(false);
      });
    return () => {
      ignore = true;
    };
  }, []);
  const save = async () => {
    setSaving(true);
    await act(
      () => api("settings", { sources, context_budget: budget }),
      "设置已保存",
    );
    setSaving(false);
  };
  return (
    <section className="standard-page settings-page">
      <div className="page-heading">
        <div>
          <h1>设置</h1>
          <p>数据的位置、接入的范围，都由你决定。</p>
        </div>
      </div>
      {settingsError && (
        <div className="inline-alert" role="alert">
          设置读取失败：{settingsError}
          <button className="quiet" onClick={() => location.reload()}>
            重新读取
          </button>
        </div>
      )}
      {settingsLoading && (
        <div className="settings-loading">
          <LoadingIndicator label="正在读取设置…" />
        </div>
      )}
      <section className="settings-section">
        <div>
          <h2>记忆位置</h2>
          <p>
            原始会话、附件、记忆和历史版本保存在这里。
            <br />
            本机访问凭据与搜索索引分开保存。
          </p>
        </div>
        <div>
          <div className="path-box">
            <Folder size={17} />
            <span>{status?.root}</span>
            <CopyButton text={status?.root ?? ""} label="复制记忆位置" />
          </div>
          <p className="field-note">
            更换位置会复制并校验完整档案，再切换本机接入。旧目录保留；同步与备份沿用你自己的方式。
          </p>
          <Disclosure label="更换记忆位置">
            <div className="path-form">
              <div className="form-field">
                <label htmlFor="new-vault-path">新的完整路径</label>
                <input
                  id="new-vault-path"
                  aria-describedby="migration-path-note"
                  placeholder="输入新目录的完整路径"
                  value={newRoot}
                  onChange={(e) => setNewRoot(e.target.value)}
                />
              </div>
              <AsyncButton
                disabled={!newRoot.trim() || newRoot.trim() === status?.root}
                busyLabel="正在迁移…"
                onAction={() =>
                  act(
                    () => api("relocate", { root: newRoot.trim() }),
                    "已开始迁移；完成后刷新页面，旧目录保留",
                  )
                }
              >
                <Folder size={16} />
                复制档案并切换位置
              </AsyncButton>
            </div>
            <p className="field-note" id="migration-path-note">
              填写本机可访问的完整目录路径。
            </p>
          </Disclosure>
        </div>
      </section>
      <section className="settings-section">
        <div>
          <h2>来源与范围</h2>
          <p>
            暂停来源只停止后续采集，
            <br />
            已经归档的资料继续保留。
          </p>
        </div>
        <div className="source-settings">
          {sources.map((s, index) => (
            <div key={s.agent}>
              <div className="source-setting-heading">
                <strong>{names[s.agent]}</strong>
                <label className="toggle">
                  <input
                    type="checkbox"
                    checked={s.enabled}
                    onChange={(e) =>
                      setSources(
                        sources.map((x, i) =>
                          i === index ? { ...x, enabled: e.target.checked } : x,
                        ),
                      )
                    }
                  />
                  <span />
                  {s.enabled ? "已启用" : "已暂停"}
                </label>
              </div>
              <label className="sr-only" htmlFor={"path-" + s.agent}>
                {names[s.agent]} 来源位置
              </label>
              <input
                id={"path-" + s.agent}
                value={s.path}
                onChange={(e) =>
                  setSources(
                    sources.map((x, i) =>
                      i === index ? { ...x, path: e.target.value } : x,
                    ),
                  )
                }
              />
            </div>
          ))}
          <AsyncButton
            className="primary"
            disabled={saving || settingsLoading || !!settingsError}
            onAction={save}
            busyLabel="正在保存…"
          >
            保存来源设置
          </AsyncButton>
        </div>
      </section>
      <section className="settings-section">
        <div>
          <h2>Agent 上下文</h2>
          <p>
            控制每次自动带入的增量正文。
            <br />
            这不会截断或删除原始档案。
          </p>
        </div>
        <div>
          <label>
            每次增量字符预算
            <input
              type="number"
              min={1024}
              max={100000}
              value={budget}
              onChange={(e) => setBudget(Number(e.target.value))}
            />
          </label>
          <AsyncButton
            onAction={save}
            busyLabel="正在保存…"
            disabled={saving || settingsLoading || !!settingsError}
          >
            保存上下文设置
          </AsyncButton>
        </div>
      </section>
      <section className="settings-section">
        <div>
          <h2>档案与恢复</h2>
          <p>
            导出包含可读原文、附件、记忆与校验清单。
            <br />
            索引可以随时从原始资料重建。
          </p>
        </div>
        <div>
          <div className="actions">
            <a className="button" href="/api/v1/export">
              <Download size={16} />
              导出完整档案
            </a>
            <button onClick={importAction}>
              <Upload size={16} />
              导入档案
            </button>
          </div>
          <div className="actions secondary-actions">
            <AsyncButton
              busyLabel="正在校验…"
              onAction={() =>
                act(async () => {
                  const v = await api<{ issues: string[] }>("repair", {
                    action: "verify",
                  });
                  if (v.issues?.length) throw Error(v.issues.join("；"));
                }, "档案校验通过")
              }
            >
              <ShieldCheck size={16} />
              校验原始资料
            </AsyncButton>
            <AsyncButton
              busyLabel="正在重建…"
              onAction={() =>
                act(
                  () => api("repair", { action: "rebuild" }),
                  "索引已从原始档案重建",
                )
              }
            >
              <RefreshCw size={16} />
              重建搜索索引
            </AsyncButton>
          </div>
        </div>
      </section>
      <section className="settings-section">
        <div>
          <h2>程序与自动运行</h2>
          <p>
            本机后台负责归档，不调用额外模型。
            <br />
            长期记忆由当前 agent 整理。
          </p>
        </div>
        <div>
          <p>Memvault {status?.version}</p>
          <p className="field-note">
            本版本为预发布版本。系统后台启动与 agent 接入由 setup
            配置；卸载默认保留档案。完整验证范围见随程序附带的说明。
          </p>
        </div>
      </section>
    </section>
  );
}
function ForgetDialog({
  kind,
  id,
  close,
  done,
}: {
  kind: string;
  id: string;
  close: () => void;
  done: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [preview, setPreview] = useState<{
    messages: number;
    sessions: string[];
    memories: string[];
    note: string;
  }>();
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    dialog.current?.showModal();
    api<typeof preview>(`forget/preview?kind=${kind}&id=${id}`)
      .then(setPreview)
      .catch((e) => setError(e.message));
    const d = dialog.current;
    const cancel = (e: Event) => {
      e.preventDefault();
      close();
    };
    d?.addEventListener("cancel", cancel);
    return () => {
      d?.removeEventListener("cancel", cancel);
      d?.close();
    };
  }, [kind, id]);
  const forget = async () => {
    setBusy(true);
    try {
      await api("forget", { kind, id, explicit: true });
      done();
    } catch (e) {
      setError((e as Error).message);
      setBusy(false);
    }
  };
  return (
    <dialog
      ref={dialog}
      className="forget-dialog"
      aria-labelledby="forget-title"
    >
      <div className="dialog-header">
        <Trash2 size={22} />
        <button className="icon-button" onClick={close} aria-label="关闭">
          <X size={18} />
        </button>
      </div>
      <h2 id="forget-title">遗忘这些本地资料？</h2>
      {preview ? (
        <>
          <p>{preview.note}</p>
          <div className="forget-counts">
            <span>{preview.sessions?.length ?? 0} 个来源会话</span>
            <span>{preview.messages} 条消息</span>
            <span>{new Set(preview.memories).size} 条关联记忆</span>
          </div>
          <p>
            这是清理档案的操作，无法从本机索引恢复。你自己的外部备份由你另行处理。
          </p>
        </>
      ) : (
        <p>正在检查影响范围…</p>
      )}
      {error && (
        <p role="alert" className="muted-warning">
          {error}
        </p>
      )}
      <div className="actions">
        <button onClick={close}>保留资料</button>
        <button
          className="danger"
          disabled={!preview || busy}
          onClick={() => void forget()}
        >
          {busy ? "正在清理…" : "遗忘这些本地资料"}
        </button>
      </div>
    </dialog>
  );
}
createRoot(document.getElementById("root")!).render(<App />);
