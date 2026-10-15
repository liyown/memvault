import { useEffect, useRef, useState } from "react";
import {
  AlertCircle,
  ArrowRight,
  ArrowUpRight,
  ChevronDown,
  ChevronRight,
  Copy,
  Download,
  FileText,
  Link,
  LoaderCircle,
  Trash2,
  Wrench,
} from "lucide-react";
import {
  api,
  date,
  names,
  sessionTitle,
  type Detail,
  type Message,
} from "./api";
import { ForgetDialog } from "./forget";
import { AgentMark, CopyButton, Empty, Prose } from "./ui";

const BATCH = 100;

function useMenu(open: boolean, close: () => void) {
  const root = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const el = root.current;
    el?.querySelector<HTMLButtonElement>('[role="menuitem"]')?.focus();
    const dismiss = (event: PointerEvent) => {
      if (!el?.contains(event.target as Node)) close();
    };
    const keyboard = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        close();
        el?.querySelector<HTMLButtonElement>('[aria-haspopup="menu"]')?.focus();
      } else if (event.key === "Tab") close();
      else if (["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) {
        const choices = Array.from(
          el?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? [],
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
    el?.addEventListener("keydown", keyboard);
    return () => {
      document.removeEventListener("pointerdown", dismiss);
      el?.removeEventListener("keydown", keyboard);
    };
  }, [open]);
  return root;
}

function firstLine(text: string) {
  const line = text.split("\n").find((l) => l.trim()) ?? "";
  return line.replace(/^[#>*\-\s`]+/, "").slice(0, 120);
}

// Archived tool results (Claude Code stores them as user turns) carry this prefix.
const TOOL_PREFIX = /^工具(?:结果| [\w.-]+)：\n?/;

function MessageView({
  m,
  agent,
  referenced,
}: {
  m: Message;
  agent: string;
  referenced: boolean;
}) {
  const tool = m.role === "tool" || TOOL_PREFIX.test(m.text);
  const text = tool ? m.text.replace(TOOL_PREFIX, "") : m.text;
  // Tool output is often long and noisy; keep it one line until asked.
  const [open, setOpen] = useState(referenced);
  const author =
    m.role === "user" && !tool
      ? "你"
      : tool
        ? "工具输出"
        : (names[agent] ?? agent);
  return (
    <article
      id={"msg-" + m.id}
      className={
        "message role-" +
        (tool ? "tool" : m.role) +
        (referenced ? " referenced" : "")
      }
    >
      <header className="message-head">
        {m.role === "user" && !tool ? (
          <span className="avatar avatar-user" aria-hidden="true">
            你
          </span>
        ) : tool ? (
          <span className="avatar avatar-tool" aria-hidden="true">
            <Wrench size={12} />
          </span>
        ) : (
          <AgentMark agent={agent} />
        )}
        <strong>{author}</strong>
        <time>{date(m.time)}</time>
        {!m.complete && <span className="flag">进行中</span>}
        <span className="message-tools">
          <CopyButton text={m.text} label="复制这条消息" size={14} />
        </span>
      </header>
      {tool && !open ? (
        <button className="tool-summary" onClick={() => setOpen(true)}>
          <ChevronRight size={14} />
          <span>{firstLine(text) || "（空输出）"}</span>
        </button>
      ) : (
        <>
          {tool && (
            <button
              className="tool-summary open"
              onClick={() => setOpen(false)}
            >
              <ChevronDown size={14} />
              <span>收起工具输出</span>
            </button>
          )}
          {tool ? (
            <pre className="tool-output">{text}</pre>
          ) : (
            <Prose text={text} />
          )}
        </>
      )}
      {m.attachments?.map((a, i) => (
        <div className="attachment" key={i}>
          <FileText size={16} />
          <span>{a.name}</span>
          {a.missing ? (
            <small>尚未取得副本</small>
          ) : (
            <a
              href={`/api/v1/objects/${a.blob}?name=${encodeURIComponent(a.name)}`}
            >
              <Download size={14} />
              下载
            </a>
          )}
        </div>
      ))}
    </article>
  );
}

export function SessionReader({
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
  const [tab, setTab] = useState<"transcript" | "summary">("transcript");
  const [offset, setOffset] = useState(0);
  const [resume, setResume] = useState(false);
  const [busy, setBusy] = useState(false);
  const [forget, setForget] = useState(false);
  const [pack, setPack] = useState("");
  const scroller = useRef<HTMLDivElement>(null);
  const resumeMenu = useMenu(resume, () => setResume(false));
  useEffect(() => {
    setOffset(0);
    setDetail(undefined);
    setPack("");
    setTab("transcript");
  }, [id, targetMessage]);
  useEffect(() => {
    let ignore = false;
    let path = `sessions/${id}?offset=${offset}&limit=${BATCH}`;
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
    }
  };
  if (!detail)
    return (
      <div className="reader-loading" role="status" aria-label="正在读取原文">
        <div />
        <div />
        <div />
      </div>
    );
  const s = detail.session;
  const summaries = detail.summaries ?? [];
  const messages = s.messages ?? [];
  const issues = s.issues ?? [];
  return (
    <>
      <div className="reader-head">
        <div className="reader-meta">
          <span className="row-source">
            <AgentMark agent={s.agent} />
            {names[s.agent] ?? s.agent}
          </span>
          <span aria-hidden="true">·</span>
          <time>{date(s.created)}</time>
          <span aria-hidden="true">·</span>
          <span>{s.count.toLocaleString()} 条消息</span>
        </div>
        <div className="reader-title-row">
          <h2 title={s.title}>{sessionTitle(s)}</h2>
          <div className="toolbar">
            <a
              className="button ghost"
              href={`/api/v1/sessions/${id}?format=markdown`}
              aria-label="导出原文"
              title="导出原文 Markdown"
            >
              <Download size={15} />
              <span className="label">导出</span>
            </a>
            <button
              className="icon-button"
              aria-label="遗忘这段会话"
              title="遗忘"
              onClick={() => setForget(true)}
            >
              <Trash2 size={16} />
            </button>
            <div className="menu-anchor" ref={resumeMenu}>
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
                <div
                  className="popover"
                  role="menu"
                  aria-label="选择接续 agent"
                >
                  {Object.entries(names).map(([key, name]) => (
                    <button
                      key={key}
                      role="menuitem"
                      onClick={() => void start(key)}
                    >
                      <AgentMark agent={key} />用 {name} 继续
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
        <div className="tabs" role="tablist">
          <button
            role="tab"
            aria-selected={tab === "transcript"}
            className={tab === "transcript" ? "active" : ""}
            onClick={() => setTab("transcript")}
          >
            对话<span className="count">{s.count.toLocaleString()}</span>
          </button>
          <button
            role="tab"
            aria-selected={tab === "summary"}
            className={tab === "summary" ? "active" : ""}
            onClick={() => setTab("summary")}
          >
            摘要
            {summaries.length > 0 && (
              <span className="count">{summaries.length}</span>
            )}
          </button>
        </div>
      </div>
      <div className="reader-scroll" ref={scroller}>
        <div className="document">
          {pack && (
            <div className="callout">
              <div>
                <strong>接续材料已准备好</strong>
                <p>目标 agent 当前不可用。复制这些材料，交给其他助手继续。</p>
              </div>
              <div className="button-row">
                <button className="ghost" onClick={() => setPack("")}>
                  收起
                </button>
                <button
                  onClick={() =>
                    void navigator.clipboard
                      .writeText(pack)
                      .then(() => notice("接续材料已复制"))
                  }
                >
                  <Copy size={15} />
                  复制
                </button>
              </div>
            </div>
          )}
          {issues.length > 0 && (
            <div className="banner tone-warning">
              <AlertCircle size={16} />
              <span>
                {s.coverage}：{issues.join("；")}
              </span>
            </div>
          )}
          {tab === "summary" ? (
            <div className="summaries">
              {summaries.length > 1 && (
                <div className="banner">
                  存在并发摘要版本，agent 会在后续讨论中合并。
                </div>
              )}
              {summaries.length ? (
                summaries.map((m) => (
                  <section className="summary" key={m.revision}>
                    <h3>{m.title}</h3>
                    <Prose text={m.body} />
                    <p className="field-note">
                      更新于 {date(m.updated)} · 已整理 {m.covered?.length ?? 0}{" "}
                      条消息
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
                  className="ghost load-more"
                  onClick={() => setOffset(Math.max(0, detail.offset - BATCH))}
                >
                  读取更早的 {Math.min(BATCH, detail.offset)} 条
                </button>
              )}
              {messages.map((m) => (
                <MessageView
                  key={m.id}
                  m={m}
                  agent={s.agent}
                  referenced={targetMessage === m.id}
                />
              ))}
              {detail.offset + messages.length < s.count && (
                <div className="button-row center">
                  <button onClick={() => setOffset(detail.offset + BATCH)}>
                    读取接下来的 {BATCH} 条
                    <ArrowRight size={15} />
                  </button>
                  <button
                    className="ghost"
                    onClick={() => setOffset(Math.max(0, s.count - BATCH))}
                  >
                    跳到最新
                  </button>
                </div>
              )}
            </div>
          )}
          {(detail.relations?.length ?? 0) > 0 && (
            <section className="relations">
              <h4>关联会话</h4>
              {detail.relations.map((r, i) => (
                <button
                  className="link-row"
                  key={i}
                  onClick={() => choose(r.from === id ? r.to : r.from)}
                >
                  <Link size={14} />
                  <span>
                    {r.kind === "continuation"
                      ? "接续"
                      : r.kind === "branch"
                        ? "分支"
                        : "引用"}{" "}
                    · {(r.from === id ? r.to : r.from).slice(0, 12)}
                  </span>
                  <ArrowUpRight size={13} />
                </button>
              ))}
            </section>
          )}
          {issues.length === 0 && <p className="archive-note">{s.coverage}</p>}
        </div>
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
