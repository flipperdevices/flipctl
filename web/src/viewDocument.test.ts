import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync } from "node:fs";
import { resolve } from "node:path";
import {
  CANONICAL_VIEW_DOCUMENT_VERSION,
  semanticText,
  viewDocumentToCanvasDrawLog,
  viewDocumentToDomSemantics,
  type ViewDocument,
} from "./viewDocumentRenderer";

const fixtureDir = resolve(__dirname, "../../schemas/view-document-fixtures");
const fixtures = readdirSync(fixtureDir)
  .filter((f) => f.endsWith(".json"))
  .map(
    (f) =>
      [
        f.replace(/\.json$/, ""),
        JSON.parse(readFileSync(resolve(fixtureDir, f), "utf8")) as ViewDocument,
      ] as const,
  );
const byId = new Map(fixtures);
const fixture = (id: string) => {
  const doc = byId.get(id);
  if (!doc) throw new Error(`missing fixture ${id}`);
  return doc;
};
const domText = (id: string) => semanticText(viewDocumentToDomSemantics(fixture(id)));
const canvasText = (id: string) =>
  viewDocumentToCanvasDrawLog(fixture(id))
    .map((c) => `${c.op}:${c.text}`)
    .join("\n");

const representativeCanvasDocs: Record<string, ViewDocument> = {
  home: {
    apiVersion: CANONICAL_VIEW_DOCUMENT_VERSION,
    sessionId: "s-canvas",
    viewId: "home",
    revision: 1,
    title: "Home",
    blocks: [
      {
        id: "apps",
        kind: "list",
        title: "Applications",
        items: [
          { label: "Network check", value: "form", description: "Open a generic command form" },
        ],
      },
    ],
    actions: [{ id: "app.open", label: "Open", enabled: true, params: { appId: "generic" } }],
  },
  form: {
    apiVersion: CANONICAL_VIEW_DOCUMENT_VERSION,
    sessionId: "s-canvas",
    viewId: "form",
    revision: 2,
    title: "Command form",
    blocks: [
      {
        id: "fields",
        kind: "form",
        title: "Inputs",
        fields: [{ id: "target", label: "Target", type: "text", required: true }],
      },
    ],
    actions: [{ id: "job.start", label: "Run", enabled: true, role: "primary" }],
  },
  running: {
    apiVersion: CANONICAL_VIEW_DOCUMENT_VERSION,
    sessionId: "s-canvas",
    viewId: "running",
    revision: 3,
    title: "Running",
    blocks: [
      {
        id: "progress",
        kind: "progress",
        title: "Progress",
        progress: { label: "Progress", value: 1, max: 4, text: "working" },
      },
      { id: "log", kind: "log", title: "Output", lines: [{ source: "stdout", text: "line one" }] },
    ],
    actions: [{ id: "job.cancel", label: "Cancel", enabled: true, role: "destructive" }],
  },
  result: {
    apiVersion: CANONICAL_VIEW_DOCUMENT_VERSION,
    sessionId: "s-canvas",
    viewId: "result",
    revision: 4,
    title: "Result",
    blocks: [
      {
        id: "summary",
        kind: "key_value",
        title: "Summary",
        pairs: [{ key: "state", value: "done" }],
      },
      {
        id: "rows",
        kind: "table",
        title: "Rows",
        columns: [{ id: "name", label: "Name" }],
        rows: [{ cells: { name: "row one" } }],
      },
    ],
    actions: [{ id: "navigation.back", label: "Back", enabled: true }],
  },
  error: {
    apiVersion: CANONICAL_VIEW_DOCUMENT_VERSION,
    sessionId: "s-canvas",
    viewId: "error",
    revision: 5,
    title: "Error",
    blocks: [{ id: "error", kind: "notice", severity: "error", text: "failed" }],
    actions: [{ id: "navigation.back", label: "Back", enabled: true }],
  },
};

describe("ViewDocument fixture contract", () => {
  it("loads the canonical generic fixture corpus", () => {
    expect(fixtures.map(([id]) => id).sort()).toEqual([
      "empty-job-history",
      "home-app-list",
      "job-history",
      "nmap-failure",
      "nmap-form",
      "nmap-running",
      "nmap-success",
      "ping-failure",
      "ping-form",
      "ping-running",
      "ping-success",
    ]);
    for (const [, doc] of fixtures) {
      expect(doc.apiVersion).toBe(CANONICAL_VIEW_DOCUMENT_VERSION);
      expect(doc.sessionId).toBeTruthy();
      expect(doc.revision).toBeGreaterThanOrEqual(0);
      expect(doc.title).toBeTruthy();
    }
  });

  it("uses generic block kinds and no app-specific public result fields", () => {
    const kinds = new Set(fixtures.flatMap(([, f]) => f.blocks.map((b) => b.kind)));
    expect([...kinds].sort()).toEqual([
      "form",
      "key_value",
      "list",
      "log",
      "notice",
      "progress",
      "table",
      "text",
    ]);
    const corpus = JSON.stringify(fixtures);
    for (const token of [
      "ping" + "Replies",
      "ping" + "Summary",
      "nmap" + "Progress",
      "nmap" + "PortStates",
    ])
      expect(corpus).not.toContain(token);
  });
});

describe("Web DOM ViewDocument semantics", () => {
  it("exposes generic home, form, running, success, and failure semantics", () => {
    expect(domText("home-app-list")).toContain("listitem:Ping:Ping ping Check host reachability");
    expect(domText("ping-form")).toContain(
      "textbox:Target:Target example.com hostname-or-ip required",
    );
    expect(domText("ping-running")).toContain("progressbar:Ping:Ping 2 of 4 replies received 2/4");
    expect(domText("ping-success")).toContain("status:Loss:Loss 0%");
    expect(domText("ping-failure")).toContain("alert:Ping failed:error partial packet loss");
    expect(domText("nmap-form")).toContain("textbox:Ports:Ports 80,443 bounded-port-list required");
    expect(domText("nmap-running")).toContain(
      "progressbar:Connect Scan:Connect Scan running against target-http 50/100",
    );
    expect(domText("nmap-success")).toContain("row:Port states:target-http 80/tcp http open");
    expect(domText("nmap-failure")).toContain(
      "alert:Nmap rejected:error target must be an allowed internal service",
    );
  });
});

describe("Web Canvas ViewDocument semantic draw log", () => {
  it("emits deterministic draw commands for every canonical fixture", () => {
    for (const [id, doc] of fixtures) {
      const commands = viewDocumentToCanvasDrawLog(doc);
      expect(commands[0]).toEqual({ op: "clear", text: "256x144" });
      expect(commands[1]).toMatchObject({ op: "title", text: doc.title, x: 8, y: 14 });
      for (const command of commands) {
        if (command.x !== undefined) expect(Number.isInteger(command.x)).toBe(true);
        if (command.y !== undefined) {
          expect(Number.isInteger(command.y)).toBe(true);
          expect(command.y).toBeGreaterThanOrEqual(0);
          expect(command.y).toBeLessThanOrEqual(144);
        }
      }
      expect(canvasText(id)).toMatchSnapshot(id);
    }
  });

  it("snapshots generic representative 256x144 states instead of PNG goldens", () => {
    for (const [state, doc] of Object.entries(representativeCanvasDocs)) {
      const commands = viewDocumentToCanvasDrawLog(doc);
      expect(commands[0]).toEqual({ op: "clear", text: "256x144" });
      expect(commands[1]).toEqual({ op: "title", text: doc.title, x: 8, y: 14 });
      expect(JSON.stringify(commands).toLowerCase()).not.toMatch(/ping|nmap/);
      for (const command of commands) {
        if (command.x !== undefined) expect(Number.isInteger(command.x)).toBe(true);
        if (command.y !== undefined) expect(Number.isInteger(command.y)).toBe(true);
      }
      expect(
        commands.map((c) => `${c.op}:${c.x ?? ""}:${c.y ?? ""}:${c.text}`).join("\n"),
      ).toMatchSnapshot(`generic-${state}`);
    }
  });
});
