---
kind: unit
title: "SLOs: the burn-rate alert that never fired"
name: slo-errorbudget-unit
---


## The situation

`checkout-api` has an SLO: 98% of requests succeed. The team even wrote
burn-rate rules — `checkout-slo`, deployed months ago, zero pages since.
Zero pages because *it cannot fire*:

```sh
kubectl -n kubelings get prometheusrule checkout-slo -o yaml
```

Two bugs are hiding in those rules. The scenery around them, all
prometheus-operator machinery:

- a **Prometheus** CR — the operator turned it into a running server
  (`prometheus-kubelings-0`)
- a **ServiceMonitor** — declares "scrape `app: checkout-api` every 15s";
  the operator compiles it into scrape config
- a **PrometheusRule** — recording rules + the alert; the operator ships it
  into Prometheus

Look at what the app *actually* exports, then at what the rules *ask for*:

```sh
kubectl get --raw "/api/v1/namespaces/kubelings/services/checkout-api:8080/proxy/metrics" | grep http_requests
# http_requests_total{code="200",method="get"} …
# http_requests_total{code="404",method="get"} …
```

**Bug 1 — a metric that doesn't exist.** The request-rate rule reads
`http_request_total` (no `s`). Prometheus doesn't error on unknown metrics
— the query just returns *empty*, forever. Same silent failure mode as the
Gatekeeper Rego bug (M6.11): observability code fails to nothing, not to
noise.

**Bug 2 — 404s burn the error budget.** The error ratio counts
`code=~"4..|5.."` — and look at the traffic: a third of it is 404s
(traffic-gen plays the scanner bot hitting `/err`). The bugged ratio reads
**~0.33**, seventeen times over the 2% budget — this alert would have paged
all quarter *about a bot*. A 404 is the client asking for something that
isn't there; a 5xx is *you* failing. The SLO's error term is 5xx only.

**Bug 3 — the one you find while fixing bug 2.** Write the numerator as
plain `sum(rate(http_requests_total{code=~"5.."}[5m]))` and — with zero
5xx in the traffic — it returns an **empty vector**, not 0. Empty divided
by anything is empty: your "fixed" ratio emits *no sample at all*, and an
alert on it can't tell "perfectly healthy" from "pipeline broken". Zero
errors must produce the number 0: `... or vector(0)`.

## Your task

1. Fix the PrometheusRule (`kubectl edit prometheusrule checkout-slo -n
   kubelings`, or re-apply):
   - `http_request_total` → `http_requests_total`
   - error-ratio numerator: `code=~"5.."` only, with an `or vector(0)`
     fallback so zero errors still emits a sample
2. Prove it against the live traffic:

   ```sh
   kubectl get --raw "/api/v1/namespaces/kubelings/services/prometheus-operated:9090/proxy/api/v1/query?query=checkout:error_ratio5m" | head -c 400
   ```

   A `"status":"success"` with a value of `0` — an honest zero — means the
   pipeline lives and the budget isn't burning.

<details>
<summary>Hint</summary>

The operator reloads rules within ~a minute of the PrometheusRule change —
`kubectl -n kubelings logs prometheus-kubelings-0 -c config-reloader
--tail=5` shows the reload. Recording rules need one evaluation interval
(~30s) before they return samples; query the raw expression first if
impatient.

</details>

::simple-task
---
:tasks: tasks
---
#active
Solve the task above — this check turns green once verification passes.

#completed
✅ Solved — nicely done!
::

<details>
<summary>Solution</summary>


## Fix

```sh
kubectl apply -n kubelings -f - <<'EOF'
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: checkout-slo
  labels: {team: kubelings}
spec:
  groups:
    - name: checkout-slo
      rules:
        - record: checkout:request_rate5m
          expr: sum(rate(http_requests_total[5m]))
        - record: checkout:error_ratio5m
          expr: |
            (sum(rate(http_requests_total{code=~"5.."}[5m])) or vector(0))
            /
            sum(rate(http_requests_total[5m]))
        - alert: CheckoutErrorBudgetBurn
          expr: checkout:error_ratio5m > 0.02
          for: 5m
          labels: {severity: page}
          annotations:
            summary: "checkout is burning error budget fast"
EOF
```

## The SLO arithmetic (why "burn rate")

98% SLO ⇒ **error budget** = 2% of requests per window. The
`error_ratio5m > 0.02` alert reads: *right now, errors arrive faster than
the budget replenishes* — burn rate > 1. Real setups alert on multiple
windows (e.g. 14× burn over 5m+1h = page; 1× over 24h = ticket) so a brief
spike and a slow leak get different responses — that's the Google SRE
multiwindow pattern, and it's why these are *recording* rules: the ratio
gets reused across windows cheaply.

The `for: 5m` matters too: ratios computed on thin traffic flap (M2.17's
lesson, one layer up the stack).

## Prevention / takeaway

- **Test alerts by firing them.** A burn-rate rule that has never paged is
  unverified code in your escalation path — trigger errors in staging and
  watch it fire, the observability twin of "test the deny AND the allow"
  (M6.11).
- Prometheus returns empty, not errors, for typo'd metrics *and* for
  filters that match nothing — lint rules (`promtool check rules`) in CI,
  alert on `absent(...)` for metrics your SLOs depend on, and make ratio
  numerators emit 0 (`or vector(0)`), never absence.
- Decide per-endpoint what counts as *your* failure: 5xx yes; 429s during
  admission-control shedding (M7.8) — a judgment call; 404s no.
- One replica, no Grafana here — the operator pattern scales the same
  YAML to real fleets; dashboards are a rendering of these exact recording
  rules.

</details>
