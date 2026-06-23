# Plugin guide

FlipCTL command plugins are declarative YAML descriptors loaded by `flipctld`. A plugin defines an app, its typed form fields, safe command mapping, parser choice, and policy hints. Plugins do **not** define HTML, TUI widgets, Canvas drawing code, frontend routes, or renderer-specific templates.

## Canonical manifest

```yaml
apiVersion: flipctl.app/v1alpha1
kind: CommandApp
metadata:
  id: network.example
  title: Example probe
  summary: Deterministic example command app
  category: Network
  version: 0.1.0
form:
  - id: target
    label: Target
    type: string
    required: true
    default: 127.0.0.1
    pattern: '^[A-Za-z0-9_.:-]+$'
  - id: count
    label: Count
    type: int
    default: "1"
    min: 1
    max: 5
execution:
  executable: ./fakecmds/fake-ping
  args: ["-c", "{{count}}", "{{target}}"]
  timeoutSeconds: 5
parser: ping
policy:
  fakeDefault: true
  network: icmp
```

The example matches the same normalized shape used by `plugins/ping.yaml` and passes `internal/plugin.Validate` after YAML loading.

## Typed fields

Each `form` field has:

- `id` — safe identifier used for template substitution and session form state.
- `label` — renderer-neutral label.
- `type` — currently `string` or `int` in the checked-in plugins.
- `required` — whether the daemon should require a value.
- `default` — string default shown in the form.
- `pattern` — optional regular expression for string validation hints.
- `min` / `max` — optional integer bounds.

## Command mapping

`execution.executable` names the binary to run and `execution.args` is an argument vector. FlipCTL uses direct exec-style invocation, not a shell. Templates such as `{{target}}` are replaced from validated form values. `timeoutSeconds` bounds the process lifetime.

Unsafe shell metacharacters in executable or argument templates fail validation. Keep commands deterministic in default plugins.

## Parser responsibility

`parser` selects daemon-side parsing for tool output, for example `ping` or `nmap`. Parsers may understand tool-specific output, but they are internal to the daemon. Parser results must be projected into generic `ViewDocument` blocks before reaching Web, TUI, or Canvas.

## Generic presentation responsibility

The daemon is responsible for translating plugin/app/job state into `text`, `notice`, `list`, `form`, `key_value`, `table`, `log`, and `progress` blocks. Frontends render those generic blocks and submit semantic session actions. A plugin must never define HTML/TUI/Canvas UI.

## Fake versus Linux command catalog

- `plugins/` is the default deterministic catalog. It uses `fakecmds/fake-ping` and `fakecmds/fake-nmap` so local tests and screenshots are reproducible without host tools or network access.
- `plugins-real/` is the opt-in Linux tools catalog for Docker `real-tools` demos. It invokes real `ping` and `nmap` inside the container and is not required by `make check`.

## Adding a command app

1. Add a YAML manifest under `plugins/` for deterministic default behavior, or under `plugins-real/` only for opt-in real tools.
2. Use `apiVersion: flipctl.app/v1alpha1` and `kind: CommandApp`.
3. Pick a stable `metadata.id` such as `network.trace`.
4. Define typed `form` fields with defaults and validation hints.
5. Map `execution.executable` and `execution.args` without shell syntax.
6. Select or add a parser if raw output needs structured treatment.
7. Project results through the existing ViewDocument model; do not add app-specific renderer fields.
8. Add tests for loading/validation, runner argument construction, parser behavior, and canonical view projection.

Useful checks:

```bash
go test ./internal/plugin ./internal/runner ./internal/api -count=1
make renderer-contract
```

## Future plugin tiers

The MVP plugin tier is declarative command wrappers. Native subprocess plugins and WebAssembly Component Model plugins are future tiers. They must preserve the same daemon-owned sessions, jobs, events, and renderer-neutral `ViewDocument` contract rather than introducing frontend-specific UI definitions.
