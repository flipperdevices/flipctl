import { describe, expect, it } from "vitest";
import {
  canonicalFieldControlId,
  canonicalFieldControlName,
  htmlPatternForValidation,
} from "./formControls";

describe("canonical form control attributes", () => {
  it("builds stable ids and names from both block id and field id", () => {
    expect(canonicalFieldControlId("form", "target")).toBe("view-field-form-target");
    expect(canonicalFieldControlId("advanced", "target")).toBe("view-field-advanced-target");
    expect(canonicalFieldControlName("form", "target")).toBe("form.target");
    expect(canonicalFieldControlName("advanced", "target")).toBe("advanced.target");
  });

  it("omits backend validation strings from HTML pattern attributes", () => {
    expect(htmlPatternForValidation(undefined)).toBeUndefined();
    expect(htmlPatternForValidation("^[A-Za-z0-9_.:/-]+$")).toBeUndefined();
    expect(htmlPatternForValidation("hostname-or-ip")).toBeUndefined();
    expect(htmlPatternForValidation("^[0-9]+$")).toBeUndefined();
  });
});
