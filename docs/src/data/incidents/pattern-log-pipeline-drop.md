---
title: "Pattern: the log pipeline that ships nothing"
description: "[PATTERN] Synthetic composite — the app logs, the agent is Ready, and the backend is empty; tail globs, parsers, rotation and backpressure all fail open and silently."
---

> **[PATTERN] scenario** — a synthetic composite of a failure mode reported
> across many production clusters. **No specific company**; details are
> representative, not cited. (Real, cited incidents are marked `[REAL]` in the
> [Incident Library](/catalog/).)

## Situation

An incident asks for last Tuesday's payment logs and there are none. Every
dashboard says the pipeline is healthy: the app is logging, the shipping agent is
`Running` with no restarts, the collector has capacity. The records simply never
arrived — and nobody can say when they stopped, because nothing ever alerted.

## Root cause chain

The defining property of log pipelines is that they **fail open, quietly**. A
tail input whose glob matches no file is not an error; there is just nothing to
tail. The agent reports itself perfectly healthy while the audit trail goes
nowhere.

| Failure | What you see | What to check |
|---|---|---|
| Glob/path mismatch | agent healthy, zero output | the agent's startup log, `Path` vs the real filename |
| Parser rejects lines | records dropped or mangled | parser config, parse-error counters |
| Rotation not followed | logs stop after a rotate | `Rotate_Wait`, position DB, copytruncate vs rename |
| Multiline stack traces | one exception → 40 useless records | multiline parser/filter |
| Backpressure | gaps only under load | `Mem_Buf_Limit`, filesystem buffering, "pausing" messages |
| Node agent misses a path | some pods only | DaemonSet mount vs the runtime's real log dir |

All of them are invisible to `kubectl get pods`. Readiness proves a process is
up; it says nothing about whether **data moved**.

A second trap rides along: agent config usually lives in a ConfigMap, and editing
a ConfigMap does not reload a running process. The fix looks applied and isn't
until the pod restarts.

## Fix & prevention

- **Monitor the pipeline's own telemetry.** Fluent Bit exposes
  `fluentbit_input_records_total`, `fluentbit_output_proc_records_total` and
  retry counters; alert on *records in ≈ 0* and on failed retries.
- **Run an end-to-end canary**: emit a heartbeat line per node per minute, alert
  when the backend stops seeing it. That one check covers the whole table.
- **Validate agent config in CI** (`fluent-bit --dry-run -c`, `vector validate`,
  `otelcol validate`) — a program that silently disagrees with your config is
  worse than one that crashes.
- **Prove the pipeline on day one.** A shipper that never shipped is
  indistinguishable from one that stopped.

## What it teaches

| Concept | Kubelings module |
|---|---|
| log shipping failure modes | M8 — `pattern-log-pipeline-drop` (drill) |
| node-level collectors | M2 — `daemonset` |
| config changes need a restart | M3 — `pattern-secret-not-reloaded` |
| telemetry pipelines & alerting on data flow | M8 — `otel-collector-pipeline`, `slo-errorbudget` |
