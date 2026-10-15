import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import {
  ArrowRight,
  Download,
  FolderInput,
  RefreshCw,
  ShieldCheck,
  Upload,
} from "lucide-react";
import { api, names, type Act, type Source, type Status } from "./api";
import { PathField, shortPath } from "./path-picker";
import { AsyncButton, LoadingIndicator, PageHeader, StatusBadge } from "./ui";

function Section({
  title,
  description,
  children,
  footer,
}: {
  title: string;
  description: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
}) {
  return (
    <section className="settings-section">
      <div className="settings-intro">
        <h2>{title}</h2>
        <p>{description}</p>
      </div>
      <div className="card settings-card">
        <div className="card-body">{children}</div>
        {footer && <div className="card-foot">{footer}</div>}
      </div>
    </section>
  );
}

const BUDGET_MIN = 1024;
const BUDGET_MAX = 100000;

export function SettingsPage({
  status,
  act,
  importAction,
}: {
  status?: Status;
  act: Act;
  importAction: () => void;
}) {
  const [newRoot, setNewRoot] = useState("");
  const [moving, setMoving] = useState(false);
  const [sources, setSources] = useState<Source[]>([]);
  const [savedSources, setSavedSources] = useState("");
  const [budget, setBudget] = useState(16000);
  const [savedBudget, setSavedBudget] = useState(16000);
  const [settingsError, setSettingsError] = useState("");
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    let ignore = false;
    api<{ sources: Source[]; context_budget: number }>("settings")
      .then((v) => {
        if (ignore) return;
        setSources(v.sources ?? []);
        setSavedSources(JSON.stringify(v.sources ?? []));
        setBudget(v.context_budget);
        setSavedBudget(v.context_budget);
      })
      .catch((e) => {
        if (!ignore) setSettingsError(e.message);
      })
      .finally(() => {
        if (!ignore) setLoading(false);
      });
    return () => {
      ignore = true;
    };
  }, []);
  const locked = loading || !!settingsError;
  const sourcesDirty = !locked && JSON.stringify(sources) !== savedSources;
  const budgetValid = budget >= BUDGET_MIN && budget <= BUDGET_MAX;
  const budgetDirty = !locked && budget !== savedBudget;
  // Both forms post the full settings so neither discards the other's saved value.
  const save = (next: { sources: Source[]; budget: number }, message: string) =>
    act(async () => {
      await api("settings", {
        sources: next.sources,
        context_budget: next.budget,
      });
      setSavedSources(JSON.stringify(next.sources));
      setSavedBudget(next.budget);
    }, message);
  const update = (index: number, patch: Partial<Source>) =>
    setSources(sources.map((x, i) => (i === index ? { ...x, ...patch } : x)));
  const target = newRoot.trim();
  const canMove = !!target && target !== status?.root;
  return (
    <div className="page page-narrow settings-page">
      <PageHeader
        title="设置"
        description="档案保存在哪里、从哪些来源采集，都由你决定。"
      />
      {settingsError && (
        <div className="banner tone-warning" role="alert">
          <span>设置读取失败：{settingsError}</span>
          <button className="ghost" onClick={() => location.reload()}>
            重新读取
          </button>
        </div>
      )}
      {loading && (
        <div className="banner">
          <LoadingIndicator label="正在读取设置…" />
        </div>
      )}

      <Section
        title="记忆位置"
        description="原始会话、附件、记忆和历史版本都在这个文件夹里。本机凭据与搜索索引另行保存。"
        footer={
          moving ? (
            <>
              <button
                className="ghost"
                onClick={() => {
                  setMoving(false);
                  setNewRoot("");
                }}
              >
                取消
              </button>
              <AsyncButton
                className="primary"
                disabled={!canMove}
                busyLabel="正在迁移…"
                onAction={() =>
                  act(async () => {
                    await api("relocate", { root: target });
                    setMoving(false);
                    setNewRoot("");
                  }, "已开始迁移；完成后刷新页面，旧目录保留")
                }
              >
                复制档案并切换
              </AsyncButton>
            </>
          ) : (
            <button onClick={() => setMoving(true)}>
              <FolderInput size={16} />
              更换位置…
            </button>
          )
        }
      >
        <div className="field">
          <span className="field-label">当前位置</span>
          <PathField label="当前记忆位置" value={status?.root ?? ""} readOnly />
        </div>
        {moving && (
          <div className="field move-field">
            <label className="field-label" htmlFor="new-vault-path">
              新位置
            </label>
            <PathField
              id="new-vault-path"
              label="选择新的记忆位置"
              prompt="选择新的记忆位置"
              value={newRoot}
              onChange={setNewRoot}
            />
            {canMove ? (
              <p className="move-plan">
                <span title={status?.root}>
                  {shortPath(status?.root ?? "", 1)}
                </span>
                <ArrowRight size={14} aria-hidden="true" />
                <strong title={target}>{shortPath(target, 2)}</strong>
              </p>
            ) : target ? (
              <p className="field-error">
                这就是当前位置，请选择另一个文件夹。
              </p>
            ) : null}
            <p className="field-note">
              会先完整复制并校验档案，再切换本机接入。旧目录原样保留，同步与备份沿用你自己的方式。
            </p>
          </div>
        )}
      </Section>

      <Section
        title="来源"
        description="暂停只停止后续采集，已经归档的资料继续保留。"
        footer={
          <>
            {sourcesDirty && <span className="dirty-note">有未保存的更改</span>}
            <AsyncButton
              className={sourcesDirty ? "primary" : ""}
              disabled={locked || !sourcesDirty}
              busyLabel="正在保存…"
              onAction={() =>
                save({ sources, budget: savedBudget }, "来源设置已保存")
              }
            >
              保存来源
            </AsyncButton>
          </>
        }
      >
        <div className="source-settings">
          {sources.map((s, index) => (
            <div className="source-setting" key={s.agent}>
              <div className="source-setting-head">
                <label className="switch">
                  <input
                    type="checkbox"
                    checked={s.enabled}
                    disabled={locked}
                    onChange={(e) =>
                      update(index, { enabled: e.target.checked })
                    }
                  />
                  <span className="switch-track" aria-hidden="true" />
                  <strong>{names[s.agent] ?? s.agent}</strong>
                </label>
                <StatusBadge tone={s.enabled ? "good" : "neutral"}>
                  {s.enabled ? "采集中" : "已暂停"}
                </StatusBadge>
              </div>
              <PathField
                id={"path-" + s.agent}
                label={`${names[s.agent] ?? s.agent} 来源位置`}
                kind={s.agent === "opencode" ? "file" : "folder"}
                value={s.path}
                disabled={locked || !s.enabled}
                onChange={(path) => update(index, { path })}
              />
            </div>
          ))}
        </div>
      </Section>

      <Section
        title="Agent 上下文"
        description="每次自动带入 agent 的增量正文上限。不影响原始档案。"
        footer={
          <AsyncButton
            className={budgetDirty && budgetValid ? "primary" : ""}
            disabled={locked || !budgetDirty || !budgetValid}
            busyLabel="正在保存…"
            onAction={() =>
              save(
                { sources: JSON.parse(savedSources || "[]"), budget },
                "上下文设置已保存",
              )
            }
          >
            保存
          </AsyncButton>
        }
      >
        <div className="field">
          <label className="field-label" htmlFor="context-budget">
            每次增量字符预算
          </label>
          <div className="input-affix">
            <input
              id="context-budget"
              type="number"
              inputMode="numeric"
              min={BUDGET_MIN}
              max={BUDGET_MAX}
              step={1000}
              value={budget}
              disabled={locked}
              aria-invalid={!budgetValid}
              aria-describedby="budget-note"
              onChange={(e) => setBudget(Number(e.target.value))}
            />
            <span>字符</span>
          </div>
          <p
            className={budgetValid ? "field-note" : "field-error"}
            id="budget-note"
          >
            范围 {BUDGET_MIN.toLocaleString()}–{BUDGET_MAX.toLocaleString()}
            ，默认 16,000。
          </p>
        </div>
      </Section>

      <Section
        title="档案与恢复"
        description="导出包含可读原文、附件、记忆与校验清单。索引随时可以从原始资料重建。"
      >
        <div className="action-group">
          <div>
            <h3>迁移</h3>
            <p className="field-note">在另一台设备或另一个位置恢复。</p>
          </div>
          <div className="button-row">
            <a className="button" href="/api/v1/export">
              <Download size={16} />
              导出完整档案
            </a>
            <button onClick={importAction}>
              <Upload size={16} />
              导入档案…
            </button>
          </div>
        </div>
        <div className="action-group">
          <div>
            <h3>维护</h3>
            <p className="field-note">检查原始资料，或重建本机搜索索引。</p>
          </div>
          <div className="button-row">
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
              校验
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
              重建索引
            </AsyncButton>
          </div>
        </div>
      </Section>

      <Section
        title="关于"
        description="本机后台负责归档，不调用额外模型；长期记忆由当前 agent 整理。"
      >
        <dl className="about-list">
          <div>
            <dt>版本</dt>
            <dd>{status?.version ?? "—"}</dd>
          </div>
          <div>
            <dt>已归档消息</dt>
            <dd>{status?.messages?.toLocaleString() ?? "—"}</dd>
          </div>
        </dl>
        <p className="field-note">
          预发布版本。后台启动与 agent 接入由 setup 配置；卸载默认保留档案。
        </p>
      </Section>
    </div>
  );
}
