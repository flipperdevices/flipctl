# Architecture demo evidence

Deterministic local daemon run using `plugins/` fake commands.

- Session W ID: `s-1`
- Session T ID: `s-2`
- Shared job ID: `j-1`
- Start action from W: `job.start` on `ping-form` revision `3`
- Cancel action from T: `job.cancel` on `ping-running` revision `2`
- Final W view: `ping-canceled` revision `5`
- Final W state evidence: `canceled`
- Schema validation success: live views are covered by `go test ./internal/api -run TestArchitecture_OneDaemonTwoSessionsOneSharedJob -count=1`; daemon served schema `https://json-schema.org/draft/2020-12/schema` during this run.
- JSON excerpt source: captured from the live projector output for the final W view in the same one-daemon/two-sessions architecture flow by running a temporary local capture test equivalent to `TestArchitecture_OneDaemonTwoSessionsOneSharedJob` and `json.MarshalIndent` on `webCanceled`; the committed regression check validates this fence against `schemas/view-document.json`.

Representative final W document excerpt:

<!-- viewdocument-json: validate captured architecture final canceled ViewDocument -->
```json
{
  "apiVersion": "viewdoc.flipctl.dev/v1alpha1",
  "sessionId": "s-1",
  "viewId": "ping-canceled",
  "revision": 5,
  "title": "Job j-1",
  "blocks": [
    {
      "id": "meta",
      "kind": "key_value",
      "title": "Job",
      "pairs": [
        {
          "key": "jobId",
          "value": "j-1"
        },
        {
          "key": "appId",
          "value": "network.ping"
        },
        {
          "key": "state",
          "value": "canceled"
        }
      ]
    },
    {
      "id": "notice",
      "kind": "notice",
      "text": "Job canceled",
      "severity": "warning"
    }
  ],
  "actions": [
    {
      "id": "navigation.back",
      "label": "Back",
      "enabled": true
    },
    {
      "id": "job.run_again",
      "label": "Run again",
      "enabled": true
    }
  ]
}
```
