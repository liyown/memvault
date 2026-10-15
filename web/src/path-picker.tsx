import { useEffect, useId, useRef, useState } from "react";
import type { KeyboardEvent } from "react";
import {
  ArrowUp,
  ChevronRight,
  File,
  Folder,
  FolderOpen,
  Home,
  X,
} from "lucide-react";
import { api } from "./api";
import { CopyButton, LoadingIndicator } from "./ui";

type Kind = "folder" | "file";
type Listing = {
  path: string;
  parent: string;
  home: string;
  separator: string;
  entries: { name: string; dir: boolean; hidden: boolean }[];
};

/** Keeps the head and the last segments of a long path readable. */
export function shortPath(path: string, keep = 2) {
  const sep = path.includes("\\") ? "\\" : "/";
  const parts = path.split(sep).filter(Boolean);
  if (parts.length <= keep + 2) return path;
  const head = sep === "/" ? sep + parts[0] : parts[0];
  return [head, "…", ...parts.slice(-keep)].join(sep);
}

/**
 * Path input with a native chooser. The local server opens the OS dialog;
 * when that is unavailable the in-app folder browser takes over.
 */
export function PathField({
  id,
  label,
  value,
  onChange,
  kind = "folder",
  prompt,
  placeholder = "选择或输入完整路径",
  readOnly,
  disabled,
}: {
  id?: string;
  label: string;
  value: string;
  onChange?: (path: string) => void;
  kind?: Kind;
  prompt?: string;
  placeholder?: string;
  readOnly?: boolean;
  disabled?: boolean;
}) {
  const fallbackId = useId();
  const inputId = id ?? fallbackId;
  const [picking, setPicking] = useState(false);
  const [browse, setBrowse] = useState(false);
  const trigger = useRef<HTMLButtonElement>(null);
  const closeBrowser = () => {
    setBrowse(false);
    // The dialog unmounts first; return focus to the button that opened it.
    requestAnimationFrame(() => trigger.current?.focus());
  };
  const [editing, setEditing] = useState(false);
  const [error, setError] = useState("");
  const choose = async () => {
    setError("");
    setPicking(true);
    try {
      const r = await api<{
        path?: string;
        cancelled?: boolean;
        unsupported?: boolean;
      }>("pick", { kind, start: value, prompt: prompt ?? label });
      if (r.path) onChange?.(r.path);
      else if (r.unsupported) setBrowse(true);
    } catch (e) {
      // A dialog already open elsewhere, or a dropped request: fall back.
      setError((e as Error).message);
      setBrowse(true);
    } finally {
      setPicking(false);
    }
  };
  return (
    <div className={`path-field ${disabled ? "is-disabled" : ""}`}>
      <div className="path-control">
        {kind === "folder" ? (
          <Folder size={16} aria-hidden="true" />
        ) : (
          <File size={16} aria-hidden="true" />
        )}
        {readOnly ? (
          <span className="path-value" id={inputId} title={value}>
            <span className="path-full">{value || "—"}</span>
          </span>
        ) : (
          <input
            id={inputId}
            aria-label={label}
            title={value}
            value={editing || !value ? value : shortPath(value, 3)}
            placeholder={placeholder}
            spellCheck={false}
            autoComplete="off"
            disabled={disabled}
            onFocus={() => setEditing(true)}
            onBlur={() => setEditing(false)}
            onChange={(e) => onChange?.(e.target.value)}
          />
        )}
        {value && <CopyButton text={value} label="复制路径" size={15} />}
        {!readOnly && (
          <button
            ref={trigger}
            className="path-choose"
            onClick={() => void choose()}
            disabled={disabled || picking}
            aria-busy={picking}
          >
            {picking ? "等待选择…" : "选择…"}
          </button>
        )}
      </div>
      {error && <p className="field-error">{error}</p>}
      {browse && (
        <FolderBrowser
          kind={kind}
          start={value}
          title={prompt ?? label}
          close={closeBrowser}
          pick={(path) => {
            onChange?.(path);
            closeBrowser();
            setError("");
          }}
        />
      )}
    </div>
  );
}

export function FolderBrowser({
  kind,
  start,
  title,
  close,
  pick,
}: {
  kind: Kind;
  start: string;
  title: string;
  close: () => void;
  pick: (path: string) => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const list = useRef<HTMLUListElement>(null);
  const [listing, setListing] = useState<Listing>();
  const [path, setPath] = useState(start);
  const [file, setFile] = useState("");
  const [hidden, setHidden] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    const d = dialog.current;
    d?.showModal();
    const cancel = (e: Event) => {
      e.preventDefault();
      close();
    };
    d?.addEventListener("cancel", cancel);
    return () => {
      d?.removeEventListener("cancel", cancel);
      d?.close();
    };
  }, []);
  useEffect(() => {
    let ignore = false;
    setLoading(true);
    api<Listing>(
      `browse?files=${kind === "file" ? 1 : 0}&path=${encodeURIComponent(path)}`,
    )
      .then((v) => {
        if (ignore) return;
        setListing(v);
        setError("");
        setFile("");
      })
      .catch((e) => {
        if (ignore) return;
        setError((e as Error).message);
        // An unreadable start location falls back to the home folder once.
        if (!listing && path) setPath("");
      })
      .finally(() => {
        if (!ignore) setLoading(false);
      });
    return () => {
      ignore = true;
    };
  }, [path, kind]);
  const sep = listing?.separator ?? "/";
  const join = (name: string) =>
    listing!.path.endsWith(sep)
      ? listing!.path + name
      : listing!.path + sep + name;
  const crumbs = (() => {
    if (!listing) return [];
    const parts = listing.path.split(sep).filter(Boolean);
    const rootless = sep === "/";
    return parts.map((name, i) => ({
      name,
      path:
        (rootless ? "/" : "") +
        parts.slice(0, i + 1).join(sep) +
        (rootless ? "" : sep),
    }));
  })();
  const entries = (listing?.entries ?? []).filter((e) => hidden || !e.hidden);
  const target = kind === "file" ? file : (listing?.path ?? "");
  const keyboard = (e: KeyboardEvent) => {
    if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(e.key)) return;
    const items = Array.from(
      list.current?.querySelectorAll<HTMLButtonElement>("button") ?? [],
    );
    if (!items.length) return;
    e.preventDefault();
    const current = items.indexOf(document.activeElement as HTMLButtonElement);
    const next =
      e.key === "Home"
        ? 0
        : e.key === "End"
          ? items.length - 1
          : Math.min(
              items.length - 1,
              Math.max(0, current + (e.key === "ArrowUp" ? -1 : 1)),
            );
    items[next]?.focus();
  };
  return (
    <dialog
      ref={dialog}
      className="dialog browser-dialog"
      aria-labelledby="browser-title"
    >
      <div className="dialog-head">
        <h2 id="browser-title">{title}</h2>
        <button className="icon-button" onClick={close} aria-label="关闭">
          <X size={18} />
        </button>
      </div>
      <div className="browser-bar">
        <button
          className="icon-button"
          aria-label="上一级"
          title="上一级"
          disabled={!listing?.parent}
          onClick={() => listing?.parent && setPath(listing.parent)}
        >
          <ArrowUp size={16} />
        </button>
        <button
          className="icon-button"
          aria-label="个人文件夹"
          title="个人文件夹"
          onClick={() => setPath(listing?.home ?? "")}
        >
          <Home size={16} />
        </button>
        <nav className="crumbs" aria-label="当前位置">
          {crumbs.map((c, i) => (
            <span key={c.path}>
              {i > 0 && <ChevronRight size={13} aria-hidden="true" />}
              <button
                aria-current={i === crumbs.length - 1 ? "location" : undefined}
                onClick={() => setPath(c.path)}
              >
                {c.name}
              </button>
            </span>
          ))}
        </nav>
      </div>
      <div className="browser-body" onKeyDown={keyboard}>
        {loading && !listing ? (
          <div className="browser-state">
            <LoadingIndicator />
          </div>
        ) : !listing ? (
          <div className="browser-state field-error">
            {error || "无法读取这个位置"}
          </div>
        ) : entries.length === 0 ? (
          <div className="browser-state">
            {kind === "file" ? "这里没有文件" : "这里没有子文件夹"}
          </div>
        ) : (
          <ul ref={list} aria-busy={loading}>
            {entries.map((e) => (
              <li key={e.name}>
                <button
                  className={file === join(e.name) ? "selected" : ""}
                  onClick={() =>
                    e.dir ? setPath(join(e.name)) : setFile(join(e.name))
                  }
                  onDoubleClick={() => !e.dir && pick(join(e.name))}
                >
                  {e.dir ? <Folder size={16} /> : <File size={16} />}
                  <span>{e.name}</span>
                  {e.dir && <ChevronRight size={14} className="trail" />}
                </button>
              </li>
            ))}
          </ul>
        )}
        {error && listing && (
          <p className="field-error browser-error">{error}</p>
        )}
      </div>
      <div className="dialog-foot">
        <label className="check">
          <input
            type="checkbox"
            checked={hidden}
            onChange={(e) => setHidden(e.target.checked)}
          />
          显示隐藏项目
        </label>
        <span className="browser-target" title={target}>
          {target
            ? shortPath(target, 3)
            : kind === "file"
              ? "尚未选择文件"
              : ""}
        </span>
        <button onClick={close}>取消</button>
        <button
          className="primary"
          disabled={!target}
          onClick={() => pick(target)}
        >
          <FolderOpen size={16} />
          {kind === "file" ? "选择此文件" : "选择此文件夹"}
        </button>
      </div>
    </dialog>
  );
}
