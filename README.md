# httpstat
`httpstat` a CLI tool that measures and visualises HTTP request performance from the terminal.

# Installation

## Homebrew (macOS / Linux)
```sh
brew install vandancd/tap/httpstat
```

## From source
```sh
go install github.com/vandancd/httpstat@latest
```

# Usage
`httpstat <url>`

## The Problem
Most developers debug HTTP performance with `curl -v` or browser DevTools. Both have serious gaps:
  - `curl -v` shows protocol events as raw text with wall-clock timestamps you have to diff manually. No visual sense of where time went.
  - Browser DevTools requires a browser useless in CI, staging servers, or scripted diagnostics.
  - Neither shows redirect chains as a unified picture, you get hop #1's timing or hop #2's timing, never both scaled together.

The mental model engineers need is: "How much of the total round-trip was DNS? Was TCP? Was the server?" — and where exactly in the redirect chain did things slow down?*

## The Waterfall Output (default, no flags needed)
`httpstat` renders a waterfall bar chart directly in the terminal. Every hop (redirect + final response) appears in order, with each phase as a proportional horizontal bar:
```
  →  http://vandan.co  301 Moved Permanently  HTTP/1.1  new connection
    DNS Lookup         ████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  7.31ms
    TCP Connection     ███░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  18.43ms
    TLS Handshake      ████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  28.96ms
    TTFB               ███░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  26.74ms

  ●  https://www.vandan.co  200 OK  HTTP/2.0  new connection
    DNS Lookup         ██░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  2.11ms
    TCP Connection     ██░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  14.27ms
    TLS Handshake      ████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░  49.84ms
    TTFB               ██████████████████████████████░░░░░░░░░░  574.21ms
    TTLB               ████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  78.56ms

    Total              ████████████████████████████████████████  716.00ms
```

## Key design decisions worth calling out:
- All bars scaled to the grand total (sum across all hops). A DNS bar in hop #1 and a TTFB bar in hop #2 are visually comparable, you can immediately see which phase dominates the whole request.
- Phase colors: DNS/TCP are cyan (network); TLS is yellow (crypto); TTFB is green/yellow/red by severity threshold (< 200ms / < 500ms / ≥ 500ms); TTLB is blue.
- → marks redirects, ● marks the final response, borrowed from network topology notation: → is a transit hop, ● is a terminal node.
- TTLB only appears on the final hop; redirects don't have response bodies, so showing it would be misleading.
- Connection reuse is called out ("reused connection"); reused connections skip DNS/TCP/TLS phases, so those rows are hidden, making it obvious why the bars look different.

## The Trace Output (`--trace` flag)
The waterfall answers "where did time go overall." The trace answers "what exactly happened, in what order, and how long between events?"
```
  Trace

    →  http://vandan.co  301 Moved Permanently
        +0ms   [+0ms]    Getting connection for vandan.co:443
        +1ms   [+1ms]    DNS lookup starting for vandan.co
        +8ms   [+7ms]    Connection attempt to 151.101.194.187:443
       +26ms  [+18ms]    TLS handshake starting
       +55ms  [+29ms]    TLS handshake completed
       +81ms  [+26ms]    First response byte received (TTFB)

    ●  https://www.vandan.co  200 OK
       +82ms   [+1ms]    Getting connection for www.vandan.co:443
       +82ms   [+0ms]    DNS lookup starting for www.vandan.co
       +84ms   [+2ms]    Connection attempt to 146.75.38.187:443
       +98ms  [+14ms]    TLS handshake starting
      +148ms  [+50ms]    TLS handshake completed
      +658ms [+510ms]    First response byte received (TTFB)
      +737ms  [+79ms]    Response body fully read (TTLB)
```

## Key design decisions:
- Two timestamp columns: +elapsed (time since probe start) and [+delta] (time since the previous event). Elapsed tells you when it happened in the overall picture; delta tells you how long that specific step took.
- Grouped by hop under the same →/● headers as the waterfall so you can read the waterfall for the summary, then look at the trace for the same hop to drill in.
- Phase-colored events: the same color palette as the waterfall bars, so DNS events are cyan, TLS events are yellow, TTFB events are severity-colored. The colors make the phase structure visible even in a flat list.
- No wall-clock timestamps, absolute timestamps are useless for debugging. What matters is relative time.

## Request Control

Control the HTTP method, headers, and body sent with each probe.

### Method (`--method` / `-X`)

Override the HTTP method. Defaults to `GET`; automatically becomes `POST` when a body flag is provided.

```sh
httpstat -X PUT https://httpbin.org/put
httpstat --method DELETE https://httpbin.org/delete
```

### Headers (`--header` / `-H`)

Add request headers. Repeatable. The same key may appear more than once — all values are sent.

```sh
httpstat -H "Accept: application/json" https://httpbin.org/get
httpstat -H "X-Request-ID: abc123" -H "X-Tenant: acme" https://httpbin.org/get

# Send two cookies
httpstat -H "Cookie: session=abc" -H "Cookie: csrf=xyz" https://httpbin.org/get
```

### Raw body (`--data` / `-d`)

Send a raw request body. No `Content-Type` is set automatically — add one with `-H` if needed. Implies `POST` when `--method` is not set.

```sh
httpstat -d "hello=world" https://httpbin.org/post
httpstat -d "hello=world" -H "Content-Type: application/x-www-form-urlencoded" https://httpbin.org/post
```

### JSON body (`--json`)

Send a JSON request body. Automatically sets `Content-Type: application/json` and `Accept: application/json`. Implies `POST` when `--method` is not set.

```sh
httpstat --json '{"user":"alice","role":"admin"}' https://httpbin.org/post

# Override Content-Type if the endpoint expects a different media type
httpstat --json '{"x":1}' -H "Content-Type: application/merge-patch+json" -X PATCH https://httpbin.org/patch
```

### Bearer token (`--bearer`)

Inject an `Authorization: Bearer <token>` header. Conflicts with `-H "Authorization: ..."`.

```sh
httpstat --bearer eyJhbGci... https://httpbin.org/bearer
```

### Conflict detection

Mutually exclusive flag combinations are caught early with a clear error:

```sh
# Error: --data and --json cannot both be set
httpstat --data "foo" --json '{"x":1}' https://httpbin.org/post

# Error: --bearer and -H "Authorization: ..." cannot both be set
httpstat --bearer tok -H "Authorization: Basic dXNlcjpwYXNz" https://httpbin.org/get

# Error: --user-agent and -H "User-Agent: ..." cannot both be set
httpstat --user-agent "MyBot/1.0" -H "User-Agent: OtherBot" https://httpbin.org/get
```

## Other Capabilities
  - `--output-json` flag outputs all timing and trace data as structured JSON, suitable for piping into jq or logging systems
  - `--dns-servers` lets you override DNS resolution to test split-horizon DNS, compare CDN PoPs, or diagnose DNS propagation
  - `--http1 / --http1.1` force protocol version (useful for comparing HTTP/1.x vs HTTP/2 performance)
  - `--ipv6` prefers IPv6 routing
  - `--no-keepalive` disables connection reuse (useful to always see the full cold-start cost)
  - `--max-redirects` controls redirect depth (default 5, range 2–10)
  - `--timeout` sets a probe-level deadline in seconds
  - `--user-agent` sets the User-Agent header (defaults to Chrome; use `--user-agent ""` to send Go's default)
  - `--version` prints the current version and exits
