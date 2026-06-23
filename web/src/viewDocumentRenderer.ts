export {
  CANONICAL_VIEW_DOCUMENT_VERSION,
  type ViewAction,
  type ViewBlock,
  type ViewDocument,
} from "./viewDocumentTypes";

import type { ViewBlock, ViewDocument } from "./viewDocumentTypes";

export type DomSemanticNode = { role: string; label: string; text: string; blockId?: string };
export type CanvasDrawCommand = {
  op: "clear" | "title" | "section" | "text" | "status";
  text: string;
  blockId?: string;
  x?: number;
  y?: number;
};

export function viewDocumentToDomSemantics(document: ViewDocument): DomSemanticNode[] {
  const nodes: DomSemanticNode[] = [
    { role: "heading", label: document.title, text: document.title },
  ];
  for (const block of document.blocks) {
    nodes.push({
      role: "region",
      label: block.title ?? block.kind,
      text: `${block.kind}: ${block.title ?? block.id}`,
      blockId: block.id,
    });
    appendBlockSemantics(nodes, block);
  }
  for (const action of document.actions ?? [])
    nodes.push({ role: "button", label: action.label, text: `${action.id} ${action.label}` });
  return nodes;
}

function assertNever(value: never): never {
  throw new Error(`unhandled ViewDocument block kind: ${JSON.stringify(value)}`);
}

function appendBlockSemantics(nodes: DomSemanticNode[], block: ViewBlock): void {
  switch (block.kind) {
    case "text":
      nodes.push({
        role: "note",
        label: block.title ?? block.kind,
        text: block.text,
        blockId: block.id,
      });
      return;
    case "notice":
      nodes.push({
        role: "alert",
        label: block.title ?? block.kind,
        text: [block.severity, block.text].join(" "),
        blockId: block.id,
      });
      return;
    case "list":
      for (const item of block.items)
        nodes.push({
          role: "listitem",
          label: item.label,
          text: [item.label, item.value, item.description].filter(Boolean).join(" "),
          blockId: block.id,
        });
      return;
    case "form":
      for (const field of block.fields)
        nodes.push({
          role: "textbox",
          label: field.label,
          text: [
            field.label,
            field.default,
            field.validation,
            field.required ? "required" : undefined,
          ]
            .filter(Boolean)
            .join(" "),
          blockId: block.id,
        });
      return;
    case "key_value":
      for (const pair of block.pairs)
        nodes.push({
          role: "status",
          label: pair.key,
          text: `${pair.key} ${pair.value}`,
          blockId: block.id,
        });
      return;
    case "table":
      for (const row of block.rows)
        nodes.push({
          role: "row",
          label: block.title ?? block.id,
          text: block.columns
            .map((c) => row.cells[c.id])
            .filter(Boolean)
            .join(" "),
          blockId: block.id,
        });
      return;
    case "log":
      for (const line of block.lines)
        nodes.push({
          role: "log",
          label: `${line.source ?? "log"} ${line.id ?? ""}`.trim(),
          text: [line.id, line.source, line.severity, line.text].filter(Boolean).join(" "),
          blockId: block.id,
        });
      return;
    case "progress":
      nodes.push({
        role: "progressbar",
        label: block.progress.label,
        text: [
          block.progress.label,
          block.progress.text,
          block.progress.value == null
            ? undefined
            : `${block.progress.value}${block.progress.max ? `/${block.progress.max}` : ""}`,
        ]
          .filter(Boolean)
          .join(" "),
        blockId: block.id,
      });
      return;
    default:
      return assertNever(block);
  }
}

export function viewDocumentToCanvasDrawLog(document: ViewDocument): CanvasDrawCommand[] {
  const commands: CanvasDrawCommand[] = [
    { op: "clear", text: "256x144" },
    { op: "title", text: document.title, x: 8, y: 14 },
  ];
  let y = 28;
  for (const node of viewDocumentToDomSemantics(document).slice(1)) {
    commands.push({
      op: node.role === "alert" ? "status" : node.role === "region" ? "section" : "text",
      text: `${node.role}:${node.label}:${node.text}`.slice(0, 80),
      blockId: node.blockId,
      x: 8,
      y,
    });
    y += 10;
    if (y > 138) break;
  }
  return commands;
}

export function semanticText(nodes: DomSemanticNode[]): string {
  return nodes.map((n) => `${n.role}:${n.label}:${n.text}`).join("\n");
}

export function canvasLinesForViewDocument(document: ViewDocument): string[] {
  const lines = [document.title];
  for (const command of viewDocumentToCanvasDrawLog(document).slice(2)) {
    const prefix = command.op === "status" ? "!" : command.op === "section" ? ">" : " ";
    lines.push(`${prefix} ${command.text}`.slice(0, 40));
  }
  if (document.actions?.length) {
    lines.push("Actions: " + document.actions.map((a) => a.label).join(" · "));
  }
  return lines.slice(0, 12);
}
