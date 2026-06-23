# Contract validation evidence

Slice A1 makes `schemas/view-document.json` a strict JSON Schema Draft 2020-12
contract for canonical `ViewDocument` payloads.

## What is validated

- All checked-in valid fixtures in `schemas/view-document-fixtures/*.json` pass
  the real Draft 2020-12 validator from `github.com/santhosh-tekuri/jsonschema/v6`.
- All targeted invalid fixtures in `schemas/invalid-view-document-fixtures/*.json`
  fail validation, including mixed block payloads, missing required block
  payloads, unknown root/block properties, unknown block kinds, and obsolete
  Ping/Nmap public fields.
- The live `GET /api/v1/sessions/{id}/view` response produced by the API test
  server validates with the same schema.

## Verification commands

Fresh A1 evidence is recorded in
`docs/plans/2026-06-22-flipctl-finalization/verification/a1-strict-schema.md`.


## A2/A3 semantic action validation

- `ViewAction` is now intent-only: required `id`, `label`, `enabled`; optional `role` and semantic string `params`.
- Schema invalid fixtures reject `method`, `path`, legacy `disabled`, and missing `enabled`.
- Canonical session actions require `viewId`, `revision`, and `actionId`; stale submissions return `409 stale_view` with current view identity/revision and do not mutate state.
