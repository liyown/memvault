import { useEffect, useId, useRef, useState } from "react";
import type {
  ButtonHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
} from "react";
import { Check, ChevronDown, Copy, LoaderCircle } from "lucide-react";

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

export function CopyButton({ text, label }: { text: string; label: string }) {
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
      {state === "copied" ? <Check size={16} /> : <Copy size={16} />}
      <span className="sr-only" role="status">
        {state !== "idle" ? feedback : ""}
      </span>
    </button>
  );
}
