# httpstat

A CLI tool for measuring and displaying HTTP request performance metrics.

## Language

**Probe**:
A single instrumented HTTP measurement operation — from URL to final response, including any redirects.
_Avoid_: request (too narrow — a probe may involve several requests), test, check

**Hop**:
One HTTP exchange within a probe. A probe involving redirects has multiple hops; the last hop is the final response.
_Avoid_: request, leg, step

**Phase**:
A named timing segment within a hop: DNS Lookup, TCP Connection, TLS Handshake, TTFB, TTLB.
_Avoid_: stage, step, metric

**TTFB** (Time To First Byte):
The duration from connection established to first response byte received. Measures server processing time.
_Avoid_: server time, response time (too vague)

**TTLB** (Time To Last Byte):
The duration of response body transfer — from first byte to body fully read. Only present on the final hop.
_Avoid_: download time, body time

**Waterfall**:
The default terminal output format. Displays each hop's phases as proportional horizontal bars, scaled relative to the grand total, stacked per hop with a total elapsed bar at the bottom.
_Avoid_: chart, diagram, visual

**TraceEvent**:
A single timestamped network event captured during a probe. Stored as raw text + time so renderers can compute elapsed and delta without parsing strings.
_Avoid_: log entry, message, log line

**Grand Total**:
The elapsed time from the start of the first hop to the end of the final hop's body read. Used to scale all waterfall bars so phases are visually comparable across hops.
_Avoid_: total time (ambiguous — each hop also has a total)

## Relationships

- A **Probe** produces one or more **Hops**; the last **Hop** is the final response
- Each **Hop** contains one or more **Phases**
- **TTLB** only appears on the final **Hop** (redirects have no body)
- The **Waterfall** scales all **Phase** bars relative to the **Grand Total**
- **TTFB** is the only **Phase** with a severity color (green < 200ms, yellow < 500ms, red ≥ 500ms)
- Each **Hop** owns its own **TraceEvents**; they are flushed at hop boundaries so the renderer can group them
- A **TraceEvent**'s elapsed is computed from the first **TraceEvent** of the **Probe** (not from wall-clock time)

## Example dialogue

> **Dev:** "Should we show TTLB for redirect hops?"
> **Domain expert:** "No — a redirect response has no body, so there's no TTLB phase. Only the final hop has TTLB."

> **Dev:** "What's the bar width for a 112ms redirect hop?"
> **Domain expert:** "It's scaled to the Grand Total — if the grand total is 716ms, a 112ms hop's bars are about 16% full."
