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
