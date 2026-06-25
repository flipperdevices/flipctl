
Plugin example:

```json
{
  "id": "ping",
  "title": "Ping Host",
  "description": "Ping utility",
  "command": "/bin/ping",
  "args": ["-c", "{count}", "{host}"],
  "cwd": "/tmp",
  "env": { "LANG": "en_US.UTF-8" },
  "timeout": 30,
  "interactive": false,
  "stopSignal": "SIGINT",
  "autoRestart": false,

  "params": [
    {
      "id": "host",
      "label": "Host",
      "type": "input",
      "default": "google.com",
      "required": true
    },
    {
      "id": "count",
      "label": "Count",
      "type": "number",
      "default": 4,
      "min": 1,
      "max": 100
    },
    {
      "id": "timeout",
      "label": "Timeout (sec)",
      "type": "number",
      "default": 2
    }
  ],

  "controls": [
    {
      "id": "status",
      "type": "label",
      "label": "Status",
      "value": "Ready"
    },
    {
      "id": "result",
      "type": "field",
      "label": "Result",
      "value": ""
    },
    {
      "id": "progress",
      "type": "progress",
      "label": "Progress",
      "value": 0,
      "min": 0,
      "max": 100
    },
    {
      "id": "runBtn",
      "type": "button",
      "label": "Run",
      "action": "execute"
    },
    {
      "id": "stopBtn",
      "type": "button",
      "label": "Stop",
      "action": "stop",
      "visibleWhen": { "paramId": "status", "value": "running" }
    },
    {
      "id": "outputTable",
      "type": "table",
      "label": "Ping responses",
      "columns": ["Time", "TTL"],
      "value": []
    }
  ],

  "parsing": {
    "rules": [
      {
        "regex": "time=([0-9.]+) ms",
        "target": "result",
        "format": "{{match}} ms",
        "stream": "stdout"
      },
      {
        "regex": "icmp_seq=([0-9]+) ttl=([0-9]+) time=([0-9.]+) ms",
        "target": "outputTable",
        "multiple": true,
        "stream": "stdout",
        "format": "{{row}}"
      },
      {
        "regex": "Destination Host Unreachable",
        "target": "status",
        "format": "Unreachable",
        "stream": "stderr"
      }
    ]
  }
}
```

## Plugin Schema

### Identification
- **`id`** (string, unique) – Task identifier.
- **`title`** (string) – Display name shown in the menu.

### Executable Command
- **`command`** (string) – Path to the executable file.
- **`args`** (array of strings) – Command‑line arguments. Supports parameter substitution using placeholders like `{paramName}`.

### User Parameters
- **`params`** (array of objects) – Describes the parameters that the user can configure via the UI.

Each parameter object has the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Parameter identifier. |
| `label` | string | Human‑readable label. |
| `type` | string | Input type: `"input"`, `"select"`, `"checkbox"`, `"number"`. |
| `default` | varies | Default value for the parameter. |
| `options` | array | (For `select` type) Array of strings or objects `{label, value}`. |
| `required` | boolean | Whether the parameter is mandatory. |

### Controls (UI Elements)
- **`controls`** (array of objects) – Defines the UI elements displayed on the task screen.

Each control object includes:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier within the screen. |
| `type` | string | Control type: `"label"`, `"field"`, `"button"`, `"input"`, `"select"`, `"progress"`, `"table"`. |
| `label` | string | (Optional) Display label. |
| `value` | varies | (For `field`) Initial value. |
| `action` | string | (For buttons) Action to perform: `"execute"`, `"stop"`, `"sendInput"`, `"refresh"`, etc. |

### Output Parsing Rules
- **`parsing`** (object) – Contains an array of `rules`.

Each rule has the following properties:

| Field | Type | Description |
|-------|------|-------------|
| `regex` | string | Regular expression pattern (as a string). |
| `target` | string | ID of the control where the matched result will be placed. |
| `format` | string | (Optional) Template for formatting, e.g., `"{{match}} ms"`. |
| `stream` | string | Which stream to process: `"stdout"`, `"stderr"`, or both (default). |
| `multiple` | boolean | If `true`, accumulates all matches (useful for tables). |

### Additional Execution Settings

| Field | Type | Description |
|-------|------|-------------|
| `cwd` | string | Working directory for the process. |
| `env` | object | Environment variables to set. |
| `timeout` | number | Maximum execution time in seconds (process is terminated after this). |
| `interactive` | boolean | Whether to use a pseudo‑terminal (PTY). |
| `stopSignal` | string | Signal to send when stopping the process (default: `"SIGTERM"`). |
| `autoRestart` | boolean | Whether to automatically restart the process after it exits (e.g., for daemons). |
| `restartDelay` | number | Delay before restarting, in seconds. |


