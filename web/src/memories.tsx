import { useEffect, useState } from "react";
import {
  ArrowLeft,
  ArrowUpRight,
  Check,
  ChevronDown,
  FileText,
  History,
  Pencil,
  Plus,
  Trash2,
} from "lucide-react";
import { api, date, kinds, type Act, type Memory } from "./api";
import { ForgetDialog } from "./forget";
import { AsyncButton, Empty, Prose, Segmented, SelectField } from "./ui";

export function Memories({
  tick,
  target,
  goSession,
  act,
}: {
  tick: number;
  target: string;
  goSession: (id: string, message?: string) => void;
  act: Act;
}) {
  const [items, setItems] = useState<Memory[]>([]);
  const [loaded, setLoaded] = useState(false);
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
    api<{ items: Memory[] }>("memories")
      .then((r) => {
        if (ignore) return;
        const next = r.items ?? [];
        setItems(next);
        if (target) {
          const found = next.find((m) => m.id === target);
          if (found) setSelected(found);
        }
      })
      .finally(() => {
        if (!ignore) setLoaded(true);
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
  const toggleHistory = async () => {
    if (!selected) return;
    if (showHistory) return setShowHistory(false);
    const r = await api<{ items: Memory[] }>(
      "memories?id=" + selected.id + "&history=true",
    );
    setHistory(r.items ?? []);
    setShowHistory(true);
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
  const visible = items.filter((m) => !filter || m.kind === filter);
  const counts = (k: string) => items.filter((m) => m.kind === k).length;
  const open = editing || !!selected;
  return (
    <>
      <div className={"split " + (open ? "has-selection" : "")}>
        <aside className="directory" aria-label="记忆目录">
          <div className="directory-head">
            <div>
              <h1>记忆</h1>
              <span className="directory-count">
                {loaded ? `${items.length} 条` : ""}
              </span>
            </div>
            <div className="toolbar">
              <button
                className="icon-button"
                aria-label="添加记忆"
                title="添加记忆"
                onClick={() => edit()}
              >
                <Plus size={17} />
              </button>
            </div>
          </div>
          <div className="directory-tools">
            <Segmented
              label="按分类筛选"
              value={filter}
              onChange={setFilter}
              options={[
                { value: "", label: `全部 ${items.length}` },
                ...Object.entries(kinds)
                  .filter(([k]) => counts(k) > 0)
                  .map(([k, label]) => ({
                    value: k,
                    label: `${label} ${counts(k)}`,
                  })),
              ]}
            />
          </div>
          <div className="directory-list">
            {visible.map((m) => (
              <button
                className={
                  "list-row " +
                  (selected?.revision === m.revision && !editing
                    ? "selected"
                    : "")
                }
                key={m.revision}
                onClick={() => choose(m)}
              >
                <span className="row-meta">
                  <span>{kinds[m.kind] ?? m.kind}</span>
                  <time>{date(m.updated)}</time>
                </span>
                <span className="row-title">{m.title}</span>
                <span className="row-excerpt">{m.body.slice(0, 120)}</span>
                {(m.pending ||
                  items.filter((x) => x.id === m.id).length > 1) && (
                  <span className="row-flags">
                    {m.pending && <span className="flag">待确认</span>}
                    {items.filter((x) => x.id === m.id).length > 1 && (
                      <span className="flag tone-warning">并发版本</span>
                    )}
                  </span>
                )}
              </button>
            ))}
            {loaded && items.length === 0 && (
              <Empty
                title="记忆会在讨论中生长"
                body="agent 会自动留下你的偏好、背景和重要结论。你也可以直接添加。"
              />
            )}
          </div>
        </aside>
        <section className="reader" aria-label="记忆内容">
          {open && (
            <button
              className="ghost mobile-back"
              onClick={() => {
                setEditing(false);
                setSelected(undefined);
              }}
            >
              <ArrowLeft size={16} />
              返回记忆列表
            </button>
          )}
          {editing ? (
            <div className="reader-scroll">
              <div className="document editor">
                <h2>{selected ? "编辑记忆" : "添加记忆"}</h2>
                <div className="editor-row">
                  <div className="field">
                    <label className="field-label" htmlFor="memory-title">
                      标题
                    </label>
                    <input
                      id="memory-title"
                      value={title}
                      onChange={(e) => setTitle(e.target.value)}
                      placeholder="这条记忆关于什么？"
                    />
                  </div>
                  <div className="field">
                    <label className="field-label" htmlFor="memory-kind">
                      分类
                    </label>
                    <SelectField
                      id="memory-kind"
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
                  </div>
                </div>
                <div className="field">
                  <label className="field-label" htmlFor="memory-body">
                    内容
                  </label>
                  <textarea
                    id="memory-body"
                    rows={16}
                    value={body}
                    onChange={(e) => setBody(e.target.value)}
                    placeholder="用清楚的文字记录，支持 Markdown。"
                  />
                </div>
                <div className="button-row end">
                  <button className="ghost" onClick={() => setEditing(false)}>
                    取消
                  </button>
                  <AsyncButton
                    className="primary"
                    disabled={!title.trim() || !body.trim()}
                    onAction={save}
                    busyLabel="正在保存…"
                  >
                    <Check size={16} />
                    保存记忆
                  </AsyncButton>
                </div>
              </div>
            </div>
          ) : selected ? (
            <>
              <div className="reader-head">
                <div className="reader-meta">
                  <span>{kinds[selected.kind] ?? selected.kind}</span>
                  <span aria-hidden="true">·</span>
                  <time>{date(selected.updated)}</time>
                  <span aria-hidden="true">·</span>
                  <span>
                    {selected.author === "user" ? "你" : selected.author}
                  </span>
                </div>
                <div className="reader-title-row">
                  <h2>{selected.title}</h2>
                  <div className="toolbar">
                    <button className="ghost" onClick={() => edit(selected)}>
                      <Pencil size={15} />
                      编辑
                    </button>
                    <button
                      className={"ghost " + (showHistory ? "pressed" : "")}
                      aria-pressed={showHistory}
                      onClick={() => void toggleHistory()}
                    >
                      <History size={15} />
                      历史
                    </button>
                    <button
                      className="icon-button"
                      aria-label="遗忘这条记忆"
                      title="遗忘"
                      onClick={() => setForget(true)}
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                </div>
              </div>
              <div className="reader-scroll">
                <article className="document">
                  <Prose text={selected.body} />
                  <section className="evidence">
                    <h4>原文依据</h4>
                    {selected.evidence?.length ? (
                      selected.evidence.map((s, i) => (
                        <button
                          className="link-row"
                          key={i}
                          onClick={() => goSession(s.session, s.message)}
                        >
                          <FileText size={15} />
                          <span>来源会话 · {s.session.slice(0, 12)}</span>
                          <ArrowUpRight size={14} />
                        </button>
                      ))
                    ) : (
                      <p className="field-note">
                        由你直接添加，尚未关联会话原文。
                      </p>
                    )}
                  </section>
                  {showHistory && (
                    <section className="history">
                      <h4>历史版本</h4>
                      <p className="field-note">
                        每个版本保留原文，恢复会创建新版本。
                      </p>
                      {history.map((m) => (
                        <details key={m.revision}>
                          <summary>
                            <span>
                              {date(m.updated)} ·{" "}
                              {m.author === "user" ? "你" : m.author}
                            </span>
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
                    </section>
                  )}
                </article>
              </div>
            </>
          ) : (
            <Empty
              title="选择一条记忆"
              body="阅读内容、追溯原文依据，或查看历史版本。"
            >
              <button onClick={() => edit()}>
                <Plus size={16} />
                添加记忆
              </button>
            </Empty>
          )}
        </section>
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
    </>
  );
}
