# ViewDocument live projector validation matrix

C2 validates live Go projector output against the checked-in Draft 2020-12 schema at `schemas/view-document.json` using `ValidateViewDocument(t, doc viewdoc.Document)` in `internal/api/viewdocument_schema_test.go`.

| State | Test evidence | Schema path |
| --- | --- | --- |
| Home app list | `TestLiveProjectorViewDocumentsValidateAgainstSchemaMatrix` | Go `viewdoc.Document` from `GET /api/v1/sessions/{id}/view` |
| Ping form | `TestLiveProjectorViewDocumentsValidateAgainstSchemaMatrix` | live session projection after canonical `app.open` |
| Ping running | `TestLiveProjectorViewDocumentsValidateAgainstSchemaMatrix` | live selected job projection while fake ping sleeps |
| Ping success | `TestLiveProjectorViewDocumentsValidateAgainstSchemaMatrix` | live selected job projection after fake ping completes |
| Ping failure | `TestLiveProjectorFailureViewDocumentsValidateAgainstSchema` | live selected job projection after failing command exits non-zero |
| Nmap form | `TestLiveProjectorViewDocumentsValidateAgainstSchemaMatrix` | live session projection after canonical `app.open` |
| Nmap running | `TestLiveProjectorViewDocumentsValidateAgainstSchemaMatrix` | live selected job projection using a slow Nmap command test app |
| Nmap success | `TestLiveProjectorViewDocumentsValidateAgainstSchemaMatrix` | live selected job projection after fake nmap completes |
| Nmap failure | `TestLiveProjectorFailureViewDocumentsValidateAgainstSchema` | live selected job projection after failing command exits non-zero |
| Job history/select | `TestLiveProjectorViewDocumentsValidateAgainstSchemaMatrix` | separate session selects an existing completed job via canonical `job.select` |
| HTTP response body | `TestLiveSessionViewEndpointValidatesAgainstViewDocumentSchema` | raw `GET /api/v1/sessions/{id}/view` response validated before decode |
| Invalid projector output | `TestValidateViewDocumentReportsInvalidProjectorOutput` | helper rejects mixed text/table block and reports schema validation failure |

Fresh checks on 2026-06-22:

- `go test ./internal/viewdoc ./internal/api -count=1` — PASS
- `make renderer-contract` — PASS
- `make check` — PASS
