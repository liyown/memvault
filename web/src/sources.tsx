import { AlertCircle, Plug, RefreshCw } from "lucide-react";
import {
  api,
  date,
  integrationState,
  names,
  sourceState,
  type Act,
  type Status,
} from "./api";
import { AgentMark, AsyncButton, PageHeader, StatusBadge } from "./ui";

export function Sources({ status, act }: { status?: Status; act: Act }) {
  const issues = status?.issues ?? [];
  return (
    <div className="page page-narrow">
      <PageHeader
        title="接入"
        description="不同 agent 共享同一份本地档案。原文先归档，摘要随后由 agent 整理。"
      >
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
          修复接入
        </AsyncButton>
        <AsyncButton
          busyLabel="正在扫描…"
          onAction={() => act(() => api("scan", {}), "已安排重新扫描")}
        >
          <RefreshCw size={16} />
          重新扫描
        </AsyncButton>
      </PageHeader>

      {issues.length > 0 && (
        <div className="banner tone-warning" role="alert">
          <AlertCircle size={16} />
          <div>
            <strong>{issues.length} 项需要处理</strong>
            <ul>
              {issues.map((x, i) => (
                <li key={i}>{x}</li>
              ))}
            </ul>
          </div>
        </div>
      )}

      <div className="card source-list">
        <div className="source-grid source-grid-head" aria-hidden="true">
          <span>来源</span>
          <span>采集</span>
          <span>自动接入</span>
          <span className="num">记录</span>
          <span>最近检查</span>
        </div>
        {Object.entries(names).map(([agent, name]) => {
          const s = status?.sources?.find((x) => x.agent === agent);
          const i = status?.integrations?.find((x) => x.agent === agent);
          const capture = s
            ? (sourceState[s.state] ?? { label: s.state, tone: "neutral" })
            : { label: "正在检测", tone: "neutral" as const };
          const hook = i
            ? (integrationState[i.state] ?? { label: i.state, tone: "neutral" })
            : undefined;
          return (
            <div className="source-grid source-row" key={agent}>
              <div className="source-name">
                <AgentMark agent={agent} />
                <strong>{name}</strong>
              </div>
              <div>
                <span className="cell-label">采集</span>
                <StatusBadge tone={capture.tone}>{capture.label}</StatusBadge>
              </div>
              <div>
                <span className="cell-label">自动接入</span>
                {hook ? (
                  <StatusBadge tone={hook.tone}>{hook.label}</StatusBadge>
                ) : (
                  <span className="muted">—</span>
                )}
              </div>
              <div className="num">
                <span className="cell-label">记录</span>
                {s?.sessions?.toLocaleString() ?? "—"}
              </div>
              <div>
                <span className="cell-label">最近检查</span>
                <time className="muted">
                  {s?.last_scan ? date(s.last_scan) : "—"}
                </time>
              </div>
              {(s?.error || i?.error) && (
                <p className="source-error">
                  <AlertCircle size={14} />
                  {s?.error || i?.error}
                </p>
              )}
            </div>
          );
        })}
      </div>
      <p className="page-note">
        归档与记忆注入相互独立。安装或修复 hook 后，需要重新打开对应
        agent；Codex 首次会要求你信任新增的
        hook。来源暂时不可用不会影响已有档案，未完成的采集会自动补做。
      </p>
    </div>
  );
}
