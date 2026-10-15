import { useCallback, useEffect, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  AlertCircle,
  Archive,
  BookOpen,
  Check,
  Folder,
  Menu,
  PanelLeftClose,
  PanelLeftOpen,
  Plug,
  RefreshCw,
  Settings,
  X,
} from "lucide-react";
import { api, type Status } from "./api";
import { Memories } from "./memories";
import { Sessions } from "./sessions";
import { SettingsPage } from "./settings";
import { Sources } from "./sources";
import { LoadingIndicator } from "./ui";
import "./style.css";

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
        <span className="brand-mark large">
          <BookOpen size={22} />
        </span>
        <LoadingIndicator label="正在打开你的记忆库…" />
      </div>
    );
  if (auth === "required")
    return (
      <div className="gate">
        <span className="brand-mark large">
          <BookOpen size={22} />
        </span>
        <h1>你的记忆，只在本机打开</h1>
        <p>在终端运行下面的命令，安全打开这个页面。</p>
        <code>memvault open</code>
        <button className="primary" onClick={() => location.reload()}>
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
  const attention =
    (status?.issues?.length ?? 0) +
    (status?.sources?.filter((s) => s.state === "attention").length ?? 0);
  const closeDrawer = () => {
    setMobile(false);
    document.querySelector<HTMLButtonElement>(".sidebar-toggle")?.focus();
  };
  const vaultName =
    status?.root.split(/[\\/]/).filter(Boolean).pop() ?? "记忆位置";
  return (
    <div
      className={`app ${mobile && narrow ? "nav-open" : ""} ${collapsed ? "nav-collapsed" : ""}`}
    >
      <button
        className="nav-scrim"
        aria-label="关闭导航遮罩"
        tabIndex={-1}
        onClick={closeDrawer}
      />
      <aside className="sidebar" id="main-navigation" inert={narrow && !mobile}>
        <div className="sidebar-head">
          <a className="brand" href="/" aria-label="Memvault 首页">
            <span className="brand-mark">
              <BookOpen size={16} strokeWidth={2} />
            </span>
            <span className="brand-name">Memvault</span>
          </a>
          <button
            className="icon-button drawer-close"
            aria-label="关闭导航"
            onClick={closeDrawer}
          >
            <X size={18} />
          </button>
          <button
            className="icon-button collapse-toggle"
            aria-label={collapsed ? "展开导航" : "收起导航"}
            title={collapsed ? "展开导航" : "收起导航"}
            aria-expanded={!collapsed}
            aria-controls="main-navigation"
            onClick={toggleSidebar}
          >
            {collapsed ? (
              <PanelLeftOpen size={17} />
            ) : (
              <PanelLeftClose size={17} />
            )}
          </button>
        </div>
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
              <Icon size={17} strokeWidth={1.8} />
              <span>{label}</span>
              {id === "sources" && attention > 0 ? (
                <span
                  className="nav-alert"
                  aria-label={`${attention} 项需要处理`}
                >
                  {attention}
                </span>
              ) : (
                count !== undefined && <small>{count.toLocaleString()}</small>
              )}
            </button>
          ))}
        </nav>
        <div className="sidebar-foot">
          <button
            className="vault-chip"
            title={status?.root}
            onClick={() => {
              setPage("settings");
              setMobile(false);
            }}
          >
            <span className="vault-icon">
              <Folder size={15} />
              <span
                className={
                  "status-dot " + (attention ? "tone-warning" : "tone-good")
                }
              />
            </span>
            <span className="vault-text">
              <strong>{vaultName}</strong>
              <small>{attention ? "有事项需要处理" : "本地运行中"}</small>
            </span>
          </button>
          <button
            className="icon-button refresh-button"
            aria-label="刷新状态"
            title="刷新状态"
            onClick={refresh}
            disabled={refreshing}
            aria-busy={refreshing}
          >
            <span className={refreshing ? "spinner" : "icon-slot"}>
              <RefreshCw size={15} />
            </span>
          </button>
        </div>
      </aside>
      <div className="workspace" inert={narrow && mobile}>
        <header className="topbar">
          <button
            className="icon-button sidebar-toggle"
            aria-label={mobile ? "收起导航" : "展开导航"}
            aria-expanded={mobile}
            aria-controls="main-navigation"
            onClick={toggleSidebar}
          >
            <Menu size={18} />
          </button>
          <span className="topbar-title">
            {nav.find((n) => n.id === page)?.label}
          </span>
          {attention > 0 && (
            <button
              className="attention-pill"
              onClick={() => setPage("sources")}
            >
              <AlertCircle size={14} />
              {attention} 项需要处理
            </button>
          )}
        </header>
        {error && (
          <div className="error-banner" role="alert">
            <AlertCircle size={16} />
            <span>{error}</span>
            <button
              className="icon-button"
              aria-label="关闭错误提示"
              onClick={() => setError("")}
            >
              <X size={15} />
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
            <Check size={15} />
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
      </div>
    </div>
  );
}
createRoot(document.getElementById("root")!).render(<App />);
