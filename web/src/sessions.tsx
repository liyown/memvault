import { useDeferredValue, useEffect, useRef, useState } from "react";
import {
  ArrowLeft,
  ArrowUpRight,
  Download,
  Search,
  Upload,
  X,
} from "lucide-react";
import { age, api, names, sessionTitle, type Hit, type Session } from "./api";
import { SessionReader } from "./reader";
import { AgentMark, Empty, LoadingIndicator, SelectField } from "./ui";

const PAGE = 50;

export function Sessions({
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
  const [query, setQuery] = useState("");
  const deferred = useDeferredValue(query.trim());
  const [agent, setAgent] = useState("");
  const [role, setRole] = useState("");
  const [sort, setSort] = useState("recent");
  const [hits, setHits] = useState<Hit[]>([]);
  const [loading, setLoading] = useState(true);
  const [offset, setOffset] = useState(0);
  const searchRef = useRef<HTMLInputElement>(null);
  const selectSession = (id: string, message = "") => {
    setMobileList(false);
    choose(id, message);
  };
  useEffect(() => {
    if (deferred) setMobileList(true);
  }, [deferred]);
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
          `search?q=${encodeURIComponent(deferred)}&agent=${agent}&role=${role}&sort=${sort}&limit=40`,
        )
      : api<{ items: Session[]; total: number }>(
          `sessions?agent=${agent}&offset=${offset}&limit=${PAGE}`,
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
  const empty = deferred ? hits.length === 0 : items.length === 0;
  return (
    <div
      className={
        "split sessions " +
        (selected ? "has-selection" : "") +
        (mobileList ? " show-mobile-list" : "")
      }
    >
      <aside className="directory" aria-label="会话目录">
        <div className="directory-head">
          <div>
            <h1>会话</h1>
            <span className="directory-count" aria-live="polite">
              {loading ? (
                <LoadingIndicator label="读取中" />
              ) : deferred ? (
                `${hits.length} 条匹配`
              ) : (
                `${total.toLocaleString()} 个`
              )}
            </span>
          </div>
          <div className="toolbar">
            <button
              className="icon-button"
              aria-label="导入档案"
              title="导入档案"
              onClick={importAction}
            >
              <Upload size={16} />
            </button>
            <a
              className="button icon-button"
              aria-label="导出记忆库"
              title="导出记忆库"
              href="/api/v1/export"
            >
              <Download size={16} />
            </a>
          </div>
        </div>
        <div className="directory-tools">
          <label className="search-box">
            <Search size={16} aria-hidden="true" />
            <input
              ref={searchRef}
              placeholder="搜索会话、原文与记忆"
              aria-label="搜索会话、原文与记忆"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
            {query ? (
              <button
                className="icon-button"
                aria-label="清空搜索"
                onClick={() => setQuery("")}
              >
                <X size={14} />
              </button>
            ) : (
              <kbd>⌘K</kbd>
            )}
          </label>
          <div className="filter-row">
            <SelectField
              className="compact"
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
                  className="compact"
                  aria-label="按角色过滤"
                  value={role}
                  onChange={(e) => setRole(e.target.value)}
                >
                  <option value="">所有角色</option>
                  <option value="user">你</option>
                  <option value="assistant">助手</option>
                  <option value="memory">记忆</option>
                </SelectField>
                <SelectField
                  className="compact"
                  aria-label="搜索结果排序"
                  value={sort}
                  onChange={(e) => setSort(e.target.value)}
                >
                  <option value="recent">按时间</option>
                  <option value="relevance">按相关度</option>
                </SelectField>
              </>
            )}
          </div>
        </div>
        <div className="directory-list">
          {loading && empty ? (
            <div
              className="skeleton-list"
              role="status"
              aria-label="正在读取会话"
            >
              {[1, 2, 3, 4, 5].map((n) => (
                <div key={n} />
              ))}
            </div>
          ) : deferred ? (
            hits.length ? (
              hits.map((h, i) => (
                <button
                  key={`${h.session}-${h.message}-${i}`}
                  className={
                    "list-row " +
                    (!h.memory && selected === h.session ? "selected" : "")
                  }
                  onClick={() =>
                    h.memory
                      ? showMemory(h.memory)
                      : h.session && selectSession(h.session, h.message)
                  }
                >
                  <span className="row-meta">
                    {h.memory ? (
                      <span className="flag">记忆</span>
                    ) : (
                      <span className="row-source">
                        <AgentMark agent={h.agent ?? ""} />
                        {names[h.agent ?? ""] ?? h.agent}
                      </span>
                    )}
                    <time>{age(h.time)}</time>
                  </span>
                  <span className="row-title">
                    {sessionTitle({
                      title: h.title,
                      agent: h.agent ?? "",
                      created: h.time,
                    })}
                  </span>
                  <span className="row-excerpt">{h.text}</span>
                  {h.memory && (
                    <span className="row-link">
                      打开记忆
                      <ArrowUpRight size={13} />
                    </span>
                  )}
                </button>
              ))
            ) : (
              <Empty
                title="没有找到匹配"
                body="试试更短的关键词，或取消来源和角色筛选。"
                icon="search"
              />
            )
          ) : items.length ? (
            items.map((s) => (
              <button
                key={s.id}
                className={"list-row " + (selected === s.id ? "selected" : "")}
                onClick={() => selectSession(s.id)}
                aria-current={selected === s.id ? "true" : undefined}
              >
                <span className="row-meta">
                  <span className="row-source">
                    <AgentMark agent={s.agent} />
                    {names[s.agent] ?? s.agent}
                  </span>
                  <time>{age(s.updated)}</time>
                </span>
                <span className="row-title" title={s.title}>
                  {sessionTitle(s)}
                </span>
                <span className="row-foot">
                  <span>{s.count.toLocaleString()} 条消息</span>
                  {s.missing > 0 && (
                    <span className="flag tone-warning">附件待补齐</span>
                  )}
                </span>
              </button>
            ))
          ) : (
            <Empty
              title="从第一次讨论开始"
              body="后台会自动发现已接入 agent 的会话，也可以导入已有的 Memvault 档案。"
            />
          )}
          {!deferred && total > PAGE && (
            <div className="pagination">
              <button
                className="ghost"
                disabled={offset === 0}
                onClick={() => setOffset(Math.max(0, offset - PAGE))}
              >
                上一页
              </button>
              <span>
                {Math.floor(offset / PAGE) + 1} / {Math.ceil(total / PAGE)}
              </span>
              <button
                className="ghost"
                disabled={offset + PAGE >= total}
                onClick={() => setOffset(offset + PAGE)}
              >
                下一页
              </button>
            </div>
          )}
        </div>
      </aside>
      <section className="reader" aria-label="会话原文">
        <button
          className="ghost mobile-back"
          onClick={() => setMobileList(true)}
        >
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
            body="从左侧选择一段会话，阅读原文、回顾摘要，或交给另一个 agent 继续。"
          />
        )}
      </section>
    </div>
  );
}
