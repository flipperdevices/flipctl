
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

| Field | Type | Description |
|-------|------|-------------|
| **`id`** | `string` (unique) | Task identifier. Must be unique across all plugins. |
| **`title`** | `string` | Display name shown in the main menu. |
| **`description`** | `string` | (Optional) Brief description of the task. |

---

### Executable Command

| Field | Type | Description |
|-------|------|-------------|
| **`command`** | `string` | Absolute or relative path to the executable file. |
| **`args`** | `array` of `string` | Command‑line arguments. Supports parameter substitution using placeholders like `{paramName}` – these are replaced with user‑provided values at runtime. |

---

### User Parameters (`params`)

An **array of objects** that define the configurable parameters displayed in the UI. Each object has the following properties:

| Field | Type | Description |
|-------|------|-------------|
| **`id`** | `string` | Unique parameter identifier (used in `args` placeholders). |
| **`label`** | `string` | Human‑readable label shown next to the input field. |
| **`type`** | `string` | Input type. Supported values: `"input"`, `"select"`, `"checkbox"`, `"number"`. |
| **`default`** | `varies` | Default value for the parameter. |
| **`options`** | `array` | (Required for `select` type) Array of strings or objects `{label, value}`. |
| **`required`** | `boolean` | Whether the parameter must be filled before execution. |
| **`min`** / **`max`** | `number` | (Optional, for `number` type) Range constraints. |

---

### Controls (`controls`)

An **array of objects** defining the UI elements (controls) displayed on the task screen. Each control object includes:

| Field | Type | Description |
|-------|------|-------------|
| **`id`** | `string` | Unique identifier within the screen. |
| **`type`** | `string` | Control type: `"label"`, `"field"`, `"button"`, `"input"`, `"select"`, `"progress"`, `"table"`. |
| **`label`** | `string` | (Optional) Display label for the control. |
| **`value`** | `varies` | Initial value (for `field`, `progress`, `table`). |
| **`action`** | `string` | (For buttons) Action to perform: `"execute"`, `"stop"`, `"sendInput"`, `"refresh"`, etc. |
| **`visibleWhen`** | `object` | (Optional) Condition to show/hide the control, e.g., `{ "paramId": "status", "value": "running" }`. |
| **`columns`** | `array` | (For `table`) Column headers. |

---

### Output Parsing Rules (`parsing.rules`)

An **array of rules** applied to the output streams of the running process. Each rule has the following properties:

| Field | Type | Description |
|-------|------|-------------|
| **`regex`** | `string` | Regular expression pattern (as a string) to match against output lines. |
| **`target`** | `string` | `id` of the control where the matched result will be placed. |
| **`format`** | `string` | (Optional) Template for formatting the captured groups, e.g., `"{{match}} ms"`. Use `{{match}}` for the full match or `{{1}}`, `{{2}}` for capture groups. |
| **`stream`** | `string` | Which stream to process: `"stdout"`, `"stderr"`, or both (if omitted, both are processed). |
| **`multiple`** | `boolean` | If `true`, all matches are accumulated (useful for tables). If `false`, only the last match is stored. |

---

### Additional Execution Settings

These top‑level fields control the runtime behaviour of the process:

| Field | Type | Description |
|-------|------|-------------|
| **`cwd`** | `string` | Working directory for the process. |
| **`env`** | `object` | Environment variables to set (key‑value pairs). |
| **`timeout`** | `number` | Maximum execution time in seconds. The process is terminated after this period. |
| **`interactive`** | `boolean` | If `true`, uses a pseudo‑terminal (PTY) via `node-pty` for interactive utilities. |
| **`stopSignal`** | `string` | Signal to send when stopping the process (default: `"SIGTERM"`). |
| **`autoRestart`** | `boolean` | Whether to automatically restart the process after it exits (useful for daemon‑like tasks). |
| **`restartDelay`** | `number` | Delay (in seconds) before restarting when `autoRestart` is `true`. |


