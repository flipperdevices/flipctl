# Public contracts

## ViewDocument source of truth

`schemas/view-document.json` is the documented source of truth for the public `ViewDocument` contract. The public TypeScript contract is mechanically generated from that schema into:

- `web/src/viewDocumentTypes.ts`

Do not edit `web/src/viewDocumentTypes.ts` by hand. It carries a generated-file banner and the source schema SHA256. Update `schemas/view-document.json` first, then regenerate and commit the generated output in the same change.

## Commands

```bash
make generate-contract   # regenerate TypeScript public contract types
make check-generated     # deterministic drift check; fails if generated output differs
```

`make check` runs `make check-generated` before the broader renderer/test/build/smoke lane, so local checks exercise contract drift. CI slice E1 should use `make check` or at least `make check-generated` in its canonical lane.

## Compatibility process

- Additive changes to existing blocks must be added to `schemas/view-document.json`, renderer handling, fixtures/tests, and regenerated TypeScript types together.
- New block kinds require schema `oneOf` entries with `kind` `const`, strict required fields, and `additionalProperties: false`; all renderers must add explicit handling before the change is compatible.
- Public TypeScript discriminants intentionally do not include `| string`; unknown block kinds should be compile-time visible and should fail schema validation until a new schema version/renderer implementation exists.
- Public `ViewAction.role` and `ViewDocument.apiVersion` are schema-derived closed literals in TypeScript for the same reason.

## Go contract handling

Go `ViewDocument` structs in `internal/viewdoc` remain hand-authored for now. The existing Draft 2020-12 validation tests validate checked-in fixtures and live API session documents against `schemas/view-document.json`, which keeps serialized Go documents aligned with the schema. Generated Go can be added later only if the generator output is high-quality and does not weaken the current validation story.
