
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


