---
kind: unit
title: "Admission: the API server edits your YAML before storing it"
name: admission-mutations-unit
---


## The situation

You've been told (7.4) that every write passes through **admission** between
authorization and etcd. Time to catch it red-handed. The init created a pod
named `bare` from a five-line spec. Diff what you wrote against what the
cluster *stored*:

```sh
kubectl -n kubelings get pod bare -o yaml | grep -E 'serviceAccount|enableServiceLinks|tolerations|kube-api-access|preemptionPolicy|priority:' -A2
```

You never wrote any of that. A `serviceAccountName: default` (6.5's automount
machinery), a projected token volume, node-lifecycle **tolerations**
(`not-ready`/`unreachable` for 300s — 8.4's five-minute eviction timer,
installed *here*), a priority (M5). Your five lines became ~200. That's the
**mutating admission chain**: in-tree plugins (ServiceAccount,
DefaultTolerationSeconds, Priority…) each get to edit the object in flight.
The webhook outage (6.3) was this same seat handed to *external* code — now
you're watching the built-ins that were there all along.

## The chain, precisely

```
authn → authz (RBAC, 6.1)
  → MUTATING admission     (edits the object: defaults, injections)
  → schema validation      (your CRD schemas, 7.5, run here)
  → VALIDATING admission   (verdict only: PSS 6.4, quota M8.3, webhooks 6.3)
  → etcd (7.3)
```

Order matters and is the exam-grade fact: **mutation first, then validation**
— validators see the *final* object, so a mutating default can be what makes
a pod pass (or fail) validation.

## Your task

Drive both halves yourself with the one in-tree policy object built for it —
a **LimitRange** named `defaults`:

1. **Mutating half** — per-container defaults: `defaultRequest` (e.g. cpu
   `50m`, memory `32Mi`) and `default` limits (e.g. cpu `500m`, memory
   `128Mi`).
2. **Validating half** — a ceiling: `max` memory `256Mi`.
3. Create a deployment `sample` (1 replica, busybox sleep, label
   `app: sample`) **with no resources block at all** — then read its *pod*
   and find the resources admission injected.
4. Watch the validating half bite: try a pod requesting `512Mi` — the API
   server refuses it at the door, same rejection class as quota (M8.3).

```sh
kubectl -n kubelings get pod -l app=sample -o jsonpath='{.items[0].spec.containers[0].resources}'
```

<details>
<summary>Hint</summary>

```yaml
apiVersion: v1
kind: LimitRange
metadata: {name: defaults, namespace: kubelings}
spec:
  limits:
    - type: Container
      defaultRequest: {cpu: 50m, memory: 32Mi}
      default: {cpu: 500m, memory: 128Mi}
      max: {memory: 256Mi}
```

Key subtlety: LimitRange applies at **pod creation** — it never edits the
Deployment. Spec stays bare; pods come out dressed. (Pods created *before*
the LimitRange keep their old shape — admission is a door, not a patrol; the
same rule you met with automount in 6.5.)

The rejection test:
`kubectl -n kubelings run oversized --image=busybox:1.36 --restart=Never -- sleep 1` with
`--overrides` requesting 512Mi — read the error message; it names the
LimitRange.

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


## Why this mechanism is everywhere

Once you see admission as "the API server's programmable doorway," half the
course reorganizes itself around it:

| Lesson | Admission role |
|---|---|
| quota rejects the 3rd pod (M8.3) | validating |
| PSS blocks the root pod (6.4) | validating |
| webhook outage (6.3) | validating, external, fail-coupled |
| SA token auto-injected (6.5) | mutating, in-tree |
| CRD schema/enum/defaults (7.5) | validation + defaulting for *your* API |
| tolerationSeconds: 300 appears (8.4) | mutating (DefaultTolerationSeconds) |
| **this lesson** | both halves, driven by you |

And the ecosystem's policy engines (Kyverno, Gatekeeper — "digests only",
"no :latest", 6.10's ladder) are *just* mutating/validating webhooks with a
rule language. There is no other mechanism; this is the whole doorway.

## Operational judgment

- **Defaults are policy, make them visible**: a LimitRange silently editing
  pods is great until someone debugs "where did 128Mi come from" for an hour.
  `kubectl describe limitrange` per namespace belongs in your triage ladder.
- **LimitRange + ResourceQuota travel together** (M8.3): quota *requires*
  requests to be set; LimitRange is what sets them for teams that forgot.
  One without the other = admission rejections at 3 a.m.
- **Mutation order gotcha**: mutating plugins run in a fixed sequence, and
  mutating *webhooks* run after in-tree ones — a webhook can undo your
  defaults. When two systems fight over a field, the audit log (6.8) is the
  referee.
- The CPU-limit debate (2.14) lands here: "no CPU limits" as *policy* means
  a LimitRange with `defaultRequest.cpu` but deliberately no `default.cpu` —
  written down, reviewable, not an accident.

</details>
