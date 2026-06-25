
State of model example:

```json
{
  "meta": {
    "version": "1.0",
    "timestamp": "2026-06-10T12:34:56.789Z"
  },
  "currentScreenId": "ping",
  "screen": {
    "id": "ping",
    "title": "Ping Host",
    "type": "app",
    "appId": "ping",
    "status": "running",
    "controls": [
      {
        "id": "status",
        "type": "label",
        "label": "Status",
        "value": "Running (4/4 packets)"
      },
      {
        "id": "host",
        "type": "input",
        "label": "Host",
        "value": "8.8.8.8",
        "default": "google.com",
        "required": true
      },
      {
        "id": "count",
        "type": "number",
        "label": "Count",
        "value": 4,
        "default": 4,
        "min": 1,
        "max": 100
      },
      {
        "id": "timeout",
        "type": "number",
        "label": "Timeout (sec)",
        "value": 2,
        "default": 2
      },
      {
        "id": "result",
        "type": "field",
        "label": "Result",
        "value": "12.3 ms"
      },
      {
        "id": "progress",
        "type": "progress",
        "label": "Progress",
        "value": 75,
        "min": 0,
        "max": 100
      },
      {
        "id": "outputTable",
        "type": "table",
        "label": "Ping responses",
        "columns": ["Time", "TTL"],
        "value": [
          ["12.3 ms", "117"],
          ["14.1 ms", "117"],
          ["11.8 ms", "117"]
        ]
      },
      {
        "id": "runBtn",
        "type": "button",
        "label": "Run",
        "action": "execute",
        "enabled": false
      },
      {
        "id": "stopBtn",
        "type": "button",
        "label": "Stop",
        "action": "stop",
        "enabled": true
      }
    ],
    "output": [
      "PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.",
      "64 bytes from 8.8.8.8: icmp_seq=1 ttl=117 time=12.3 ms",
      "64 bytes from 8.8.8.8: icmp_seq=2 ttl=117 time=14.1 ms",
      "64 bytes from 8.8.8.8: icmp_seq=3 ttl=117 time=11.8 ms"
    ],
    "error": null
  },
  "navigation": {
    "canGoBack": true,
    "stackSize": 2
  }
}
```
