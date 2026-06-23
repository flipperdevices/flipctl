const sanitizeAttributePart = (value: string): string =>
  value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_-]+/g, "-")
    .replace(/^-+|-+$/g, "") || "field";

export function canonicalFieldControlId(blockId: string, fieldId: string): string {
  return `view-field-${sanitizeAttributePart(blockId)}-${sanitizeAttributePart(fieldId)}`;
}

export function canonicalFieldControlName(blockId: string, fieldId: string): string {
  return `${sanitizeAttributePart(blockId)}.${sanitizeAttributePart(fieldId)}`;
}

export function htmlPatternForValidation(_validation?: string): string | undefined {
  // Canonical ViewDocument field.validation is backend/API validation metadata and may be
  // either a backend regexp or a semantic validation token. Do not mirror it into HTML
  // pattern; backend validation remains authoritative and browsers avoid /v-regexp warnings.
  return undefined;
}
