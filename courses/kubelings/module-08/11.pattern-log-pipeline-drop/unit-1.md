---
kind: unit
title: "Drill — the log pipeline that ships nothing"
name: pattern-log-pipeline-drop-unit
---


> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters (tail globs that match nothing, parsers that reject
> every line, rotation the agent never follows), not a specific company's
> incident.

## The situation

Someone opens an incident: *"we have no payment logs for last Tuesday."* You go
looking, and every dashboard you own says the pipeline is fine.

```sh
kubectl -n kubelings get pods -l app=payments        # 2/2 Running, no restarts
kubectl -n kubelings logs deploy/payments -c app --tail=3      # the app is logging
kubectl -n kubelings logs deploy/payments -c shipper --tail=20 # …the shipper isn't
```

The `payments` pod runs the vendor application plus a **fluent-bit sidecar** that
tails the app's log directory and forwards records. The app writes. The shipper
is Ready. Nothing arrives.

This is the log pipeline's defining property: **it fails open, quietly.** A tail
input whose glob matches no file is not an error — there is simply nothing to
tail, and the agent reports itself perfectly healthy while your audit trail goes
to `/dev/null`.

<!-- d2:tail -->
```text
  ┌───────────────────┐   
  │app writes app.log │   
  │                   │   
  └───────────────────┘   
           │              
     shared volume        
           │              
           ▼              
 ┌───────────────────────┐
 │tail glob wants *.json │
 │                       │
 └───────────────────────┘
           │              
    matches no file       
           │              
           ▼              
 ┌─────────────────────┐  
 │zero records shipped │  
 │                     │  
 └─────────────────────┘  
```
<!-- /d2:tail -->

## Your task

Get records flowing again — without touching the application:

1. The app keeps writing `/var/log/app/app.log`. That path belongs to the vendor;
   changing it is not the fix.
2. The shipper's `stdout` OUTPUT stays (that's the "backend" in this drill).
3. `kubectl -n kubelings logs deploy/payments -c shipper` must show the app's
   records — look for `order_id`.

```sh
kubectl -n kubelings exec deploy/payments -c app -- ls -l /var/log/app
kubectl -n kubelings get configmap shipper-config -o jsonpath='{.data.fluent-bit\.conf}'
```

<details>
<summary>Hint</summary>

Compare what the app writes with what the `[INPUT]` glob matches. Then remember
that a ConfigMap edit does **not** reload a running process (M3
`pattern-secret-not-reloaded`, same trap):

```sh
kubectl -n kubelings patch configmap shipper-config --type=merge -p "$(cat <<'EOF'
{"data": {"fluent-bit.conf": "[SERVICE]\n    Flush        1\n    Log_Level    info\n    Daemon       Off\n[INPUT]\n    Name         tail\n    Path         /var/log/app/*.log\n    Tag          payments.*\n    Read_from_Head On\n    Refresh_Interval 5\n[OUTPUT]\n    Name         stdout\n    Match        payments.*\n    Format       json_lines\n"}}
EOF
)"
kubectl -n kubelings rollout restart deploy/payments
kubectl -n kubelings rollout status  deploy/payments
kubectl -n kubelings logs deploy/payments -c shipper --tail=10
```

</details>

::simple-task
---
:tasks: tasks
:name: verify_done
---
#active
Solve the task above — this check turns green once verification passes.

#completed
✅ Solved — nicely done!
::

<details>
<summary>Solution</summary>

## Root cause

`Path /var/log/app/*.json` matched nothing; the app writes `app.log`. Fluent-bit
started, loaded the input, found zero files, and waited politely forever.

## The pattern: how log pipelines lose records

| Failure | What you see | What to check |
|---|---|---|
| **Glob mismatch** (this drill) | agent healthy, zero output | agent's own log at startup, `Path` vs the real filename |
| **Parser rejects the line** | records dropped or mangled | `Parser`/`Parser_Firstline`, agent's parse-error counter |
| **Rotation not followed** | logs stop after a rotate | `Rotate_Wait`, `DB` position file, copytruncate vs rename |
| **Multiline stack traces** | one exception = 40 useless records | multiline filter / `Multiline.parser` |
| **Backpressure** | gaps under load only | `Mem_Buf_Limit`, `storage.type filesystem`, agent "pausing" messages |
| **Node-level agent misses a path** | some pods only | DaemonSet mount vs the runtime's real log dir |

Every one of them is invisible to `kubectl get pods`. Readiness proves a process
is up. It says nothing about whether *data* moved.

## Fix

Point the input at the file the app actually writes, then restart the pod so the
new ConfigMap is read. Config in a ConfigMap is only "live" if the process
watches it — fluent-bit does not reload on file change by default.

## Prevention / takeaway

- **Monitor the pipeline's own telemetry, not just its liveness.** Fluent-bit
  exposes `/api/v1/metrics/prometheus` (`fluentbit_input_records_total`,
  `fluentbit_output_proc_records_total`, `..._retries_failed_total`). Alert on
  *records in ≈ 0* and on retry failures — the same class of alert as an SLO burn
  rate (M8 `slo-errorbudget`).
- **End-to-end canary:** emit a heartbeat log line per node every minute and
  alert when the backend stops seeing it. That single check catches the whole
  table above.
- **Treat the agent config as code** — it is parsed by a program that will not
  tell you it disagreed. A CI job running `fluent-bit --dry-run -c` against the
  rendered config catches typos before the cluster swallows them.
- **A shipper that has never shipped is indistinguishable from one that stopped.**
  Assert the pipeline works on day one, not on the day you need last Tuesday.

</details>
