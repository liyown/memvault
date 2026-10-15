import { useEffect, useRef, useState } from "react";
import { Trash2, X } from "lucide-react";
import { api } from "./api";

type Preview = {
  messages: number;
  sessions: string[];
  memories: string[];
  note: string;
};

export function ForgetDialog({
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
  const [preview, setPreview] = useState<Preview>();
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    dialog.current?.showModal();
    api<Preview>(`forget/preview?kind=${kind}&id=${id}`)
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
      className="dialog forget-dialog"
      aria-labelledby="forget-title"
    >
      <div className="dialog-head">
        <span className="dialog-icon tone-danger">
          <Trash2 size={18} />
        </span>
        <button className="icon-button" onClick={close} aria-label="关闭">
          <X size={18} />
        </button>
      </div>
      <div className="dialog-body">
        <h2 id="forget-title">遗忘这些本地资料？</h2>
        {preview ? (
          <>
            <p>{preview.note}</p>
            <dl className="forget-counts">
              <div>
                <dt>来源会话</dt>
                <dd>{preview.sessions?.length ?? 0}</dd>
              </div>
              <div>
                <dt>消息</dt>
                <dd>{preview.messages}</dd>
              </div>
              <div>
                <dt>关联记忆</dt>
                <dd>{new Set(preview.memories).size}</dd>
              </div>
            </dl>
            <p className="field-note">
              清理后无法从本机索引恢复。你自己的外部备份需另行处理。
            </p>
          </>
        ) : (
          <p>正在检查影响范围…</p>
        )}
        {error && (
          <p role="alert" className="field-error">
            {error}
          </p>
        )}
      </div>
      <div className="dialog-foot">
        <button onClick={close}>保留资料</button>
        <button
          className="danger"
          disabled={!preview || busy}
          onClick={() => void forget()}
        >
          {busy ? "正在清理…" : "遗忘"}
        </button>
      </div>
    </dialog>
  );
}
