import { useEffect, useId, useRef, useState } from "react";
import type {
  ButtonHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
} from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import {
  BookOpen,
  Check,
  ChevronDown,
  Copy,
  LoaderCircle,
  Search,
} from "lucide-react";
import { names, type Tone } from "./api";

export function PageHeader({
  title,
  description,
  children,
}: {
  title: string;
  description?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <header className="page-header">
      <div>
        <h1>{title}</h1>
        {description && <p>{description}</p>}
      </div>
      {children && <div className="page-actions">{children}</div>}
    </header>
  );
}

export function StatusBadge({
  tone = "neutral",
  children,
}: {
  tone?: Tone;
  children: ReactNode;
}) {
  return (
    <span className={`status-badge tone-${tone}`}>
      <span className="status-dot" aria-hidden="true" />
      {children}
    </span>
  );
}

export function AgentMark({ agent }: { agent: string }) {
  return (
    <span className={`agent-mark agent-${agent}`} aria-hidden="true">
      {(names[agent] ?? agent ?? "?").slice(0, 1)}
    </span>
  );
}

export function Segmented<T extends string>({
  value,
  options,
  onChange,
  label,
}: {
  value: T;
  options: { value: T; label: string }[];
  onChange: (value: T) => void;
  label: string;
}) {
  return (
    <div className="segmented" role="radiogroup" aria-label={label}>
      {options.map((o) => (
        <button
          key={o.value}
          role="radio"
          aria-checked={value === o.value}
          className={value === o.value ? "active" : ""}
          onClick={() => onChange(o.value)}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}

export function Prose({ text }: { text: string }) {
  return (
    <div className="prose">
      <Markdown
        remarkPlugins={[remarkGfm]}
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

export function Empty({
  title,
  body,
  icon = "book",
  children,
}: {
  title: string;
  body: string;
  icon?: "book" | "search";
  children?: ReactNode;
}) {
  const Icon = icon === "book" ? BookOpen : Search;
  return (
    <div className="empty">
      <span className="empty-icon">
        <Icon size={22} strokeWidth={1.6} />
      </span>
      <h3>{title}</h3>
      <p>{body}</p>
      {children && <div className="empty-actions">{children}</div>}
    </div>
  );
}

export function SelectField({
  children,
  className = "",
  ...props
}: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <span className={`select-field ${className}`}>
      <select {...props}>{children}</select>
      <span className="select-chevron" aria-hidden="true">
        <ChevronDown />
      </span>
    </span>
  );
}

export function LoadingIndicator({ label = "正在读取…" }: { label?: string }) {
  return (
    <span className="loading-indicator" role="status">
      <span className="spinner" aria-hidden="true">
        <LoaderCircle />
      </span>
      {label}
    </span>
  );
}

export function AsyncButton({
  children,
  onAction,
  busyLabel = "正在处理…",
  className = "",
  disabled,
  ...props
}: Omit<ButtonHTMLAttributes<HTMLButtonElement>, "onClick"> & {
  onAction: () => Promise<unknown>;
  busyLabel?: string;
}) {
  const [pending, setPending] = useState(false);
  const running = useRef(false);
  return (
    <button
      {...props}
      type={props.type ?? "button"}
      className={`${className} async-button ${pending ? "is-pending" : ""}`}
      disabled={disabled || pending}
      aria-busy={pending}
      onClick={async () => {
        if (running.current) return;
        running.current = true;
        setPending(true);
        try {
          await onAction();
        } finally {
          running.current = false;
          setPending(false);
        }
      }}
    >
      <span className="button-content" aria-hidden={pending || undefined}>
        {children}
      </span>
      {pending && (
        <span className="button-progress">
          <span className="spinner" aria-hidden="true">
            <LoaderCircle />
          </span>
          <span>{busyLabel}</span>
        </span>
      )}
    </button>
  );
}

export function Disclosure({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const id = useId();
  return (
    <div className={`disclosure ${open ? "is-open" : ""}`}>
      <button
        className="disclosure-trigger"
        aria-expanded={open}
        aria-controls={id}
        onClick={() => setOpen(!open)}
      >
        <span className="disclosure-chevron" aria-hidden="true">
          <ChevronDown />
        </span>
        {label}
      </button>
      <div id={id} className="disclosure-region" inert={!open}>
        <div>
          <div className="disclosure-content">{children}</div>
        </div>
      </div>
    </div>
  );
}

export function CopyButton({
  text,
  label,
  size = 16,
}: {
  text: string;
  label: string;
  size?: number;
}) {
  const [state, setState] = useState<"idle" | "copied" | "failed">("idle");
  useEffect(() => {
    if (state === "idle") return;
    const timer = setTimeout(() => setState("idle"), 2000);
    return () => clearTimeout(timer);
  }, [state]);
  const feedback =
    state === "copied"
      ? "已复制"
      : state === "failed"
        ? "复制失败，请重试"
        : label;
  return (
    <button
      className="icon-button copy-button"
      title={feedback}
      aria-label={feedback}
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text);
          setState("copied");
        } catch {
          setState("failed");
        }
      }}
    >
      {state === "copied" ? <Check size={size} /> : <Copy size={size} />}
      <span className="sr-only" role="status">
        {state !== "idle" ? feedback : ""}
      </span>
    </button>
  );
}
