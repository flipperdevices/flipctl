import { render } from "preact";
import { useEffect, useMemo, useRef, useState } from "preact/hooks";
import {
  canonicalFieldControlId,
  canonicalFieldControlName,
  htmlPatternForValidation,
} from "./formControls";
import {
  canvasLinesForViewDocument,
  type ViewAction,
  type ViewBlock,
  type ViewDocument,
} from "./viewDocumentRenderer";

const api = async <T,>(path: string, init?: RequestInit): Promise<T> => {
  const response = await fetch("/api/v1" + path, init);
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    const error = new Error(formatProblem(body)) as Error & { status?: number; code?: string };
    error.status = response.status;
    error.code = body?.error?.code;
    throw error;
  }
  return body as T;
};

const styles = `
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color:#e8edf2;background:#091018}body{margin:0}main{max-width:1240px;margin:0 auto;padding:24px}.hero{display:flex;gap:18px;align-items:flex-start;justify-content:space-between;margin-bottom:18px}.hero h1{margin:0;font-size:34px}.subtitle{color:#9fb1c7;max-width:760px}.grid{display:grid;grid-template-columns:1fr 1fr;gap:18px}.card{background:#101b29;border:1px solid #203047;border-radius:16px;padding:16px;box-shadow:0 12px 32px #0005}.card h2,.card h3{margin:0 0 10px}.view-block{background:#0b1420;border:1px solid #21334b;border-radius:14px;padding:14px;margin:12px 0}.field{display:grid;gap:4px;margin:10px 0}.field span{font-weight:650}.field input,.field select{background:#07111d;color:#f8fbff;border:1px solid #36516f;border-radius:10px;padding:10px}.hint{font-size:12px;color:#90a5bd}.button-row{display:flex;flex-wrap:wrap;gap:8px;align-items:center}.primary{background:#52d273;color:#06110b;border:none;border-radius:10px;padding:10px 14px;font-weight:800;cursor:pointer}.secondary{background:#22334a;color:#eaf3ff;border:1px solid #3a5575;border-radius:10px;padding:9px 12px;cursor:pointer}.danger{background:#4b1d25;color:#ffccd4;border:1px solid #963749;border-radius:10px;padding:9px 12px}.primary:disabled,.secondary:disabled,.danger:disabled{opacity:.5;cursor:not-allowed}.badge{display:inline-block;border-radius:999px;padding:4px 9px;font-size:12px;font-weight:800}.badge.info{background:#233d68;color:#99c7ff}.badge.success{background:#173d29;color:#8ff0ad}.badge.warning{background:#4d3b18;color:#ffe27a}.badge.error{background:#4f1f29;color:#ffadbc}.list{display:grid;gap:8px;margin:0;padding:0;list-style:none}.list li{background:#08111c;border:1px solid #1f3148;border-radius:10px;padding:10px}.list-item-main{display:flex;gap:10px;align-items:center;justify-content:space-between}.kv-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:10px}.kv{background:#08111c;border:1px solid #1f3148;border-radius:10px;padding:10px}.table{width:100%;border-collapse:collapse;margin-top:8px}.table th,.table td{text-align:left;border-bottom:1px solid #26384f;padding:6px}.log{background:#050a10;border:1px solid #1d2d42;border-radius:12px;padding:10px;max-height:260px;overflow:auto;font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px}.log-line{white-space:pre-wrap;margin:2px 0}.stream{color:#71d0ff;margin-right:6px}.progress-track{height:10px;background:#07111d;border:1px solid #36516f;border-radius:999px;overflow:hidden}.progress-fill{height:100%;background:#52d273}.canvas-wrap{display:grid;place-items:center;background:#050a10;border-radius:14px;padding:14px}.canvas-wrap canvas{image-rendering:pixelated;width:512px;max-width:100%;height:auto;border:1px solid #3b536f}.note{background:#1d2b17;color:#d6ffc7;border:1px solid #4c7d3b;border-radius:12px;padding:10px;margin-bottom:14px}.empty{color:#91a4ba}@media(max-width:900px){.grid{grid-template-columns:1fr}.hero{display:block}}`;

type Session = { sessionId: string; revision: number };
type ApiEvent = { type: string; sessionId?: string; revision?: number };
type FormValues = Record<string, string>;

function formatProblem(body: any): string {
  const err = body?.error || body;
  const field = err?.field ? ` (${err.field})` : "";
  return `${err?.code || "request_failed"}: ${err?.message || "request failed"}${field}`;
}

function defaultFormValues(document?: ViewDocument): FormValues {
  const values: FormValues = {};
  for (const block of document?.blocks ?? []) {
    for (const field of block.fields ?? []) values[field.id] = field.default ?? "";
  }
  return values;
}

function actionValues(action: ViewAction, formValues: FormValues): FormValues {
  return { ...formValues, ...action.params };
}

function ActionBar({
  document,
  formValues,
  onAction,
}: {
  document: ViewDocument;
  formValues: FormValues;
  onAction: (action: ViewAction, values?: FormValues) => void;
}) {
  return (
    <div class="button-row" aria-label="View actions">
      {(document.actions ?? []).map((action) => {
        const isPrimary = action.role === "primary" || action.id === "job.start";
        const isDanger = action.role === "destructive" || action.id === "job.cancel";
        return (
          <button
            key={action.id}
            type="button"
            class={isDanger ? "danger" : isPrimary ? "primary" : "secondary"}
            disabled={!action.enabled}
            onClick={() => onAction(action, actionValues(action, formValues))}
          >
            {action.label}
          </button>
        );
      })}
    </div>
  );
}

function BlockRenderer({
  block,
  values,
  onField,
  onAction,
}: {
  block: ViewBlock;
  values: FormValues;
  onField: (field: string, value: string) => void;
  onAction: (action: ViewAction) => void;
}) {
  if (block.kind === "text")
    return (
      <section class="view-block">
        <p>{block.text}</p>
      </section>
    );
  if (block.kind === "notice")
    return (
      <section class="view-block" role={block.severity === "error" ? "alert" : "note"}>
        <span class={`badge ${block.severity || "info"}`}>{block.severity || "info"}</span>
        <p>{block.text}</p>
      </section>
    );
  if (block.kind === "list")
    return (
      <section class="view-block">
        <h3>{block.title}</h3>
        <ul class="list">
          {(block.items ?? []).map((item) => (
            <li key={`${item.label}-${item.value}`}>
              <div class="list-item-main">
                <span>
                  <strong>{item.label}</strong>
                  {item.value ? <span> · {item.value}</span> : null}
                </span>
                {item.action ? (
                  <button
                    type="button"
                    class="secondary"
                    disabled={!item.action.enabled}
                    onClick={() => onAction(item.action!)}
                  >
                    {item.action.label}
                  </button>
                ) : null}
              </div>
              {item.description ? <p class="hint">{item.description}</p> : null}
            </li>
          ))}
        </ul>
      </section>
    );
  if (block.kind === "form")
    return (
      <section class="view-block">
        <h3>{block.title}</h3>
        {(block.fields ?? []).map((field) => {
          const controlId = canonicalFieldControlId(block.id, field.id);
          const controlName = canonicalFieldControlName(block.id, field.id);
          const htmlPattern = htmlPatternForValidation(field.validation);
          return (
            <label class="field" htmlFor={controlId} key={field.id}>
              <span>{field.label}</span>
              {field.options?.length ? (
                <select
                  id={controlId}
                  name={controlName}
                  value={values[field.id] ?? field.default ?? ""}
                  required={field.required}
                  onChange={(e) => onField(field.id, (e.target as HTMLSelectElement).value)}
                >
                  {field.options.map((option) => (
                    <option key={option} value={option}>
                      {option}
                    </option>
                  ))}
                </select>
              ) : (
                <input
                  id={controlId}
                  name={controlName}
                  aria-label={field.label}
                  type={field.type === "int" || field.type === "number" ? "number" : "text"}
                  required={field.required}
                  pattern={htmlPattern}
                  value={values[field.id] ?? field.default ?? ""}
                  onInput={(e) => onField(field.id, (e.target as HTMLInputElement).value)}
                />
              )}
              {field.validation ? <small class="hint">{field.validation}</small> : null}
            </label>
          );
        })}
      </section>
    );
  if (block.kind === "key_value")
    return (
      <section class="view-block">
        <h3>{block.title}</h3>
        <div class="kv-grid">
          {(block.pairs ?? []).map((pair) => (
            <div class="kv" key={pair.key}>
              <strong>{pair.key}</strong>
              <br />
              {pair.value}
            </div>
          ))}
        </div>
      </section>
    );
  if (block.kind === "table")
    return (
      <section class="view-block">
        <h3>{block.title}</h3>
        <table class="table">
          <thead>
            <tr>
              {(block.columns ?? []).map((c) => (
                <th key={c.id}>{c.label}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {(block.rows ?? []).map((row, i) => (
              <tr key={i}>
                {(block.columns ?? []).map((c) => (
                  <td key={c.id}>{row.cells[c.id] ?? ""}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    );
  if (block.kind === "log")
    return (
      <section class="view-block">
        <h3>{block.title}</h3>
        <div class="log">
          {(block.lines ?? []).map((line, i) => (
            <div class="log-line" key={line.id ?? i}>
              <span class="stream">{line.source ?? "log"}</span>
              {line.text}
            </div>
          ))}
        </div>
      </section>
    );
  if (block.kind === "progress") {
    const p = block.progress;
    const pct =
      p?.value != null && p.max ? Math.max(0, Math.min(100, (p.value / p.max) * 100)) : 100;
    return (
      <section class="view-block">
        <h3>{block.title ?? p?.label ?? "Progress"}</h3>
        <div
          class="progress-track"
          role="progressbar"
          aria-label={p?.label}
          aria-valuenow={p?.value}
          aria-valuemax={p?.max}
        >
          <div class="progress-fill" style={{ width: `${pct}%` }} />
        </div>
        <p class="hint">
          {[
            p?.label,
            p?.text,
            p?.value != null ? `${p.value}${p.max ? `/${p.max}` : ""}` : undefined,
          ]
            .filter(Boolean)
            .join(" · ")}
        </p>
      </section>
    );
  }
  return (
    <section class="view-block">
      <h3>{block.title ?? block.kind}</h3>
      <pre class="log">{JSON.stringify(block, null, 2)}</pre>
    </section>
  );
}

function CanvasPanel({
  document,
  onAction,
}: {
  document?: ViewDocument;
  onAction: (action: ViewAction) => void;
}) {
  const ref = useRef<HTMLCanvasElement>(null);
  const lines = useMemo(
    () =>
      document ? canvasLinesForViewDocument(document) : ["FlipCTL", "Waiting for session view"],
    [document],
  );
  useEffect(() => {
    const canvas = ref.current;
    if (!canvas) return;
    const x = canvas.getContext("2d")!;
    x.fillStyle = "#071018";
    x.fillRect(0, 0, 256, 144);
    x.font = "10px monospace";
    lines.slice(0, 11).forEach((line, i) => {
      x.fillStyle = i === 0 ? "#70ff9c" : line.startsWith("!") ? "#ff8ca3" : "#d7e7f7";
      x.fillText(line.slice(0, 38), 8, 14 + i * 12);
    });
  }, [lines]);
  const actions = document?.actions ?? [];
  return (
    <section class="card">
      <h2>Canvas renderer preview</h2>
      <p class="hint">
        Same current ViewDocument as Web, drawn deterministically into an exact 256×144 preview.
        Controls below are simulator/browser controls outside the canvas surface.
      </p>
      <div class="canvas-wrap">
        <canvas
          width={256}
          height={144}
          ref={ref}
          aria-label="256x144 canonical ViewDocument renderer"
        />
      </div>
      <div class="button-row" aria-label="Canvas simulator/browser controls">
        {actions.slice(0, 4).map((action) => (
          <button
            key={action.id}
            class={action.id === "job.cancel" ? "danger" : "secondary"}
            type="button"
            disabled={!action.enabled}
            onClick={() => onAction(action)}
          >
            Simulator: {action.label}
          </button>
        ))}
      </div>
      <p class="hint" aria-live="polite">
        {document ? `${document.viewId} · rev ${document.revision}` : "No view loaded"}
      </p>
    </section>
  );
}

function App() {
  const [session, setSession] = useState<Session>();
  const [document, setDocument] = useState<ViewDocument>();
  const [formValues, setFormValues] = useState<FormValues>({});
  const [notice, setNotice] = useState("Creating session…");
  const [events, setEvents] = useState<ApiEvent[]>([]);

  const refreshView = async (sessionId = session?.sessionId) => {
    if (!sessionId) return;
    const next = await api<ViewDocument>(`/sessions/${sessionId}/view`);
    setDocument(next);
    setFormValues(defaultFormValues(next));
    setNotice(`Loaded ${next.title} · revision ${next.revision}`);
  };

  useEffect(() => {
    let active = true;
    api<Session>("/sessions", { method: "POST" })
      .then(async (created) => {
        if (!active) return;
        setSession(created);
        const next = await api<ViewDocument>(`/sessions/${created.sessionId}/view`);
        if (!active) return;
        setDocument(next);
        setFormValues(defaultFormValues(next));
        setNotice(`Session ${created.sessionId} ready`);
      })
      .catch((e) => setNotice(`session failed: ${e}`));
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (!session?.sessionId) return;
    const es = new EventSource("/api/v1/events");
    const onViewChanged = (message: MessageEvent) => {
      const event = JSON.parse(message.data) as ApiEvent;
      setEvents((prev) => [event, ...prev].slice(0, 40));
      if (event.sessionId === session.sessionId)
        refreshView(session.sessionId).catch((e) => setNotice(`refresh failed: ${e}`));
    };
    es.addEventListener("view.changed", onViewChanged as EventListener);
    es.onerror = () => setNotice("SSE disconnected; retrying automatically…");
    return () => {
      es.removeEventListener("view.changed", onViewChanged as EventListener);
      es.close();
    };
  }, [session?.sessionId]);

  const setField = (field: string, value: string) =>
    setFormValues((prev) => ({ ...prev, [field]: value }));

  const invoke = async (action: ViewAction, values = formValues) => {
    if (!session || !document) return;
    setNotice(`Running ${action.label}…`);
    try {
      const next = await api<ViewDocument | unknown>(`/sessions/${session.sessionId}/actions`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          actionId: action.id,
          viewId: document.viewId,
          revision: document.revision,
          values,
          fields: values,
          params: action.params,
        }),
      });
      if ((next as ViewDocument).blocks) {
        setDocument(next as ViewDocument);
        setFormValues(defaultFormValues(next as ViewDocument));
      } else await refreshView(session.sessionId);
      setNotice(`${action.label} accepted`);
    } catch (e) {
      if ((e as Error & { status?: number }).status === 409) {
        await refreshView(session.sessionId);
        setNotice("View changed; refreshed latest view before retry.");
      } else setNotice(`${action.label} failed: ${e}`);
    }
  };

  return (
    <main>
      <style>{styles}</style>
      <section class="hero">
        <div>
          <h1>FlipCTL live demo</h1>
          <p class="subtitle">
            Canonical session/view/action frontend using generic ViewDocument blocks for Web DOM and
            the deterministic 256×144 Canvas renderer preview.
          </p>
        </div>
        <button class="secondary" type="button" onClick={() => refreshView()}>
          Refresh view
        </button>
      </section>
      <div class="note" aria-live="polite">
        {notice}
      </div>
      <section class="grid">
        <section class="card">
          <h2>{document?.title ?? "Loading canonical view"}</h2>
          {document ? (
            <>
              <p class="hint">
                {document.viewId} · session {document.sessionId} · revision {document.revision}
              </p>
              {document.blocks.map((block) => (
                <BlockRenderer
                  key={block.id}
                  block={block}
                  values={formValues}
                  onField={setField}
                  onAction={invoke}
                />
              ))}
              <ActionBar document={document} formValues={formValues} onAction={invoke} />
            </>
          ) : (
            <p class="empty">Waiting for session view…</p>
          )}
        </section>
        <CanvasPanel document={document} onAction={(action) => invoke(action)} />
      </section>
      <section class="card" style="margin-top:18px">
        <h2>Canonical view events</h2>
        {events.length ? (
          events.map((event, i) => (
            <p key={i} class="hint">
              {event.type} · {event.sessionId} · revision {event.revision}
            </p>
          ))
        ) : (
          <p class="empty">No view changes yet.</p>
        )}
      </section>
    </main>
  );
}

render(<App />, document.getElementById("app")!);
