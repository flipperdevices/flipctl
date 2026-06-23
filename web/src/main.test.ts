import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { canvasLinesForViewDocument, type ViewDocument } from "./viewDocumentRenderer";

const read = (path: string) => readFileSync(resolve(__dirname, path), "utf8");

describe("canonical Web live render paths", () => {
  it("uses standards mode in the Web HTML entrypoint", () => {
    expect(read("../index.html").toLowerCase()).toMatch(/^<!doctype html>/);
  });

  it("keeps public Web entrypoint on generic ViewDocument rendering", () => {
    const entrypoint = read("main.tsx");
    for (const forbidden of [
      "pingReplies",
      "pingSummary",
      "nmapProgress",
      "nmapPortStates",
      "nmapPorts",
      'appId.includes("ping")',
      'appId.includes("nmap")',
      'api<AppSummary[]>("/apps")',
      'api("/apps")',
      'api("/jobs")',
      '"/api/v1/apps"',
      '"/api/v1/jobs"',
      '"/api/v1/actions"',
      '"/api/v1/views"',
      '"/cancel"',
      "Application switcher",
    ]) {
      expect(entrypoint).not.toContain(forbidden);
    }
    expect(entrypoint).toContain("/sessions");
    expect(entrypoint).toContain("view.changed");
    expect(entrypoint).toContain("/actions");
    expect(entrypoint).toContain("item.action");
    expect(entrypoint).toContain("Canvas renderer preview");
    expect(entrypoint).toContain("simulator/browser controls");
  });

  it("keeps Canvas documentation scoped to renderer preview evidence", () => {
    const readRoot = (path: string) => read(resolve(__dirname, "../..", path));
    const docs = ["README.md", "docs/architecture.md", "docs/demo.md"].map(readRoot).join("\n");
    expect(docs).toContain("What Canvas proves");
    expect(docs).toContain("What Canvas does not yet prove");
    expect(docs).toMatch(/Canvas renderer preview|renderer preview/);
    expect(docs).not.toMatch(/Canvas is a fully interactive embedded frontend/i);
  });

  it("wires canonical form controls with browser-safe attributes", () => {
    const entrypoint = read("main.tsx");
    expect(entrypoint).toContain("htmlFor={controlId}");
    expect(entrypoint).toContain("id={controlId}");
    expect(entrypoint).toContain("name={controlName}");
    expect(entrypoint).toContain("htmlPatternForValidation(field.validation)");
    expect(entrypoint).not.toContain("pattern={field.validation || undefined}");
  });

  it("builds compact canvas lines from canonical block/action semantics", () => {
    const doc: ViewDocument = {
      apiVersion: "viewdoc.flipctl.dev/v1alpha1",
      sessionId: "s-test",
      viewId: "generic-result",
      revision: 7,
      title: "Generic result",
      blocks: [
        { id: "n", kind: "notice", severity: "success", text: "complete" },
        { id: "kv", kind: "key_value", title: "Facts", pairs: [{ key: "state", value: "done" }] },
        {
          id: "t",
          kind: "table",
          title: "Rows",
          columns: [{ id: "name", label: "Name" }],
          rows: [{ cells: { name: "row one" } }],
        },
      ],
      actions: [{ id: "navigation.back", label: "Back", enabled: true }],
    };
    expect(canvasLinesForViewDocument(doc).join("\n")).toContain("Generic result");
    expect(canvasLinesForViewDocument(doc).join("\n")).toContain("Actions: Back");
  });
});
