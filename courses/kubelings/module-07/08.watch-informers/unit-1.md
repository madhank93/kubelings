---
kind: unit
title: "Watch: how every controller hears the cluster"
name: watch-informers-unit
---


> **Reading with live experiments.** No verify — but every section has a
> command to run on your cluster. Open two shells (`t` in the TUI, twice) and
> actually run them; this mechanism is invisible until you've seen it stream.

## The question this answers

The reconcile loop (7.1) reacts to changes "within seconds." The scheduler
(7.2) notices new pods instantly. kubectl `-w` updates live. **How?** Nothing
polls — polling at Kubernetes scale would melt etcd. The answer is one
mechanism used by literally every component: **list + watch**.

## See it raw

Terminal 1 — subscribe to the change feed:

```sh
kubectl -n kubelings get pods --watch --output-watch-events
```

Terminal 2 — cause changes:

```sh
kubectl -n kubelings create deploy echo --image=busybox:1.36 -- sleep 3600
kubectl -n kubelings scale deploy echo --replicas=3
kubectl -n kubelings delete deploy echo
```

Terminal 1 streams `ADDED` / `MODIFIED` / `DELETED` events *as they happen* —
several MODIFIED per pod as status converges (scheduled → containerCreating →
running: each a status write, each an event). This is not kubectl magic; it's
an HTTP GET with `?watch=true` that the API server holds open, fed from
etcd's own watch stream (7.3).

## resourceVersion: the cursor

Every object and every LIST response carries a `resourceVersion` — a cluster
-wide, monotonically-ordered position in the change log:

```sh
kubectl -n kubelings get deploy -o jsonpath='{.metadata.resourceVersion}{"\n"}'
```

The pattern every client uses:

```
LIST   → full state as of resourceVersion N   (expensive, once)
WATCH  ?watch=true&resourceVersion=N          (cheap, forever)
         → stream of exactly the changes after N
```

Two consequences worth keeping:

- **Nothing is missed and nothing is polled.** The watch resumes precisely
  where the list stopped. If the connection drops, re-watch from the last
  seen version.
- **"410 Gone" and the relist**: the API server only keeps a bounded window
  of history. A client that reconnects with an ancient resourceVersion gets
  `410 Gone` and must re-LIST — the expensive path. Periodic `BOOKMARK`
  events exist purely to keep clients' cursors fresh so this stays rare.
  A *relist storm* — thousands of clients re-listing at once after an API
  server restart — is a classic control-plane brownout (and one ingredient
  of the OpenAI outage, 9.2).

## Informers: the cache that makes controllers cheap

A controller written naively — GET on every reconcile — would hammer the API
server. Real controllers (via client-go's **informer**) do:

```
one LIST+WATCH per resource type
  → local in-memory cache (always current, event-driven)
  → your reconcile reads from the CACHE, free
  → events also enqueue keys onto a workqueue (dedup, retry, backoff)
```

That's why a cluster runs *hundreds* of controllers without the API server
noticing: reads are local; only the watch connection and actual writes touch
the wire. It's also why controller memory scales with object *count* — the
cache holds everything it watches (a fleet-scale gotcha: a DaemonSet agent
informing on all pods holds all pods in RAM, on every node — one more
ingredient of 9.2).

**Level-triggered, the design soul:** the workqueue stores *keys*, not events.
Reconcile reads the cache's *current* state and converges — so a missed
event, a crash, a dropped watch all heal on the next sync. Kubernetes never
asks "what happened?"; it asks "what should the world look like *now*?" That
one choice is why the platform is crash-safe all the way down (7.1's loop,
now with its plumbing visible).

## APF: the API server defends itself

All those watchers and listers share one front door. **API Priority &
Fairness** is the bouncer — requests are classified into flow schemas and
priority levels, each with its own concurrency budget:

```sh
kubectl get flowschemas
kubectl get prioritylevelconfigurations
```

Read the built-in ladder: `system-leader-election` outranks
`workload-high` outranks `global-default` outranks `catch-all`. Meaning: when
the API server saturates (9.2!), kubelet heartbeats and leader-election
renewals keep flowing while someone's runaway list-everything client gets
queued or shed (429s). It's quota logic (M8.3) applied to the control plane's
own front door — and the reason a modern cluster degrades instead of
collapsing under one bad client. (Tuning it is rare; *knowing it exists*
turns "the API is slow" from mystery into a `kubectl get flowschemas` away
from an answer.)

## What to carry out of this

- `--watch --output-watch-events` is a debugging tool, not a curiosity —
  watching a namespace during a rollout shows you the exact write sequence.
- Any tool you build against the API: LIST once, WATCH after, never poll.
- Controller memory ∝ watched-object count; agent-per-node × watch-everything
  = fleet incident.
- API slow? Check APF rejection/queue metrics before blaming etcd.

*No check — run the experiments, then advance: the next lesson builds the
controller these primitives exist for.*
