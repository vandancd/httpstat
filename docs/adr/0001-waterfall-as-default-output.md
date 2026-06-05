# Waterfall is the default output; JSON requires --json

The tool's primary audience is developers reading output in a terminal, not machines consuming JSON. The default output is a human-readable waterfall showing phases as proportional colored bars. JSON output requires `--json`; trace messages require `--trace`. This is the opposite of the previous behaviour where JSON was always emitted.

## Considered options

- **JSON by default** — machine-readable, easy to pipe, but hard to scan at a glance for "why was this slow"
- **Waterfall by default** — human-readable, instant visual diagnosis, JSON available on demand via `--json`

Chose waterfall because the tool is used interactively to diagnose slow URLs, not as a pipeline component. A user who needs JSON knows to ask for it; a user who needs a fast answer should not have to parse JSON to get it.
