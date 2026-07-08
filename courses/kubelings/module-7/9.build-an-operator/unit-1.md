---
kind: unit
title: "Build an operator: the verb for your noun"
name: build-an-operator-unit
---


> **Reading — internals capstone.** Writing and deploying real Go is outside
> the lesson sandbox, but after 7.1 (the loop), 7.5 (the CRD), and 7.8 (watch)
> you own every concept; this assembles them into the ~60 lines that make an
> operator an operator. Read the code slowly — there is nothing else to it.

## Where we left off

Lesson 7.5 ended with a confession: your `BackupSchedule` CRs are **inert
data** — `nightly-etcd` says "back up at 2 a.m." and nothing on earth reads
it. The missing piece is a controller. Not a framework, not magic:
a program that runs the 7.1 loop against *your* kind using the 7.8 plumbing.

## The whole shape, one diagram

```
        (7.8: one LIST+WATCH on BackupSchedules, one on CronJobs)
                            │ events
                            ▼
                     ┌──────────────┐
                     │  workqueue   │   keys like "kubelings/nightly-etcd"
                     │ dedup/retry/ │   NOT events — level-triggered!
                     │   backoff    │
                     └──────┬───────┘
                            ▼
              reconcile(key) — the ONLY logic you write
                            │
              read desired (CR) + observed (cache) → make ONE move → requeue
```

## The reconcile, annotated

Go-flavored pseudocode; every line maps to a lesson you've done:

```go
func reconcile(key string) error {
    // 1. Fetch DESIRED from the informer cache (7.8) — free, no API call.
    sched, found := cache.Get(key)
    if !found {
        // CR deleted. Do we clean up? NO — see ownerReferences below.
        return nil
    }

    // 2. Fetch OBSERVED: does the child CronJob (2.5) already exist?
    want := buildCronJob(sched)          // pure function: CR -> child spec
    have, exists := cronjobCache.Get(childKey(sched))

    // 3. Converge — exactly one of three moves, then let the loop re-run:
    switch {
    case !exists:
        // ownerReference: the ONE line that buys garbage collection —
        // delete the CR and the cluster deletes the CronJob for you (3.5's
        // finalizer machinery, working in your favor this time).
        want.OwnerReferences = []metav1.OwnerReference{ownerOf(sched)}
        return client.Create(want)

    case !specEqual(have, want):
        return client.Update(merge(have, want))

    default:
        // 4. Reality matches desire → report it. Status is the controller's
        //    ONLY output besides children (7.1: spec is yours, status is mine).
        return client.UpdateStatus(sched, lastRun(have), "Ready")
    }
}
```

Three properties make this production-grade rather than a toy, and all three
are *inherited from the pattern*, not written by you:

- **Idempotent**: reconcile computes moves from current state; running it
  twice is harmless. This is mandatory, because the queue *will* hand you
  duplicates.
- **Crash-safe**: kill the controller anywhere — on restart, LIST rebuilds
  the cache, every key gets re-reconciled, the world heals (the same
  level-triggered argument from 7.8; Monzo's 9.1 lesson is what happens when
  a system is edge-triggered instead).
- **Collision-safe**: writes carry `resourceVersion`; a conflicting write
  fails with 409, the key requeues with backoff, reconcile re-reads and
  retries. No locks anywhere.

Two more habits complete the checklist: **leader election** (7.4) so two
replicas of your controller don't fight — a Lease and a flag in
controller-runtime; and **RBAC** (6.1) scoped to exactly what reconcile
touches (your CRD + CronJobs, nothing else — the audit-friendly identity
story from 6.5).

## The ladder (what to actually use in 2026)

| Rung | What it gives | When |
|---|---|---|
| raw `client-go` | informers/workqueue by hand (what you just read) | never for real work; once, to learn |
| **`controller-runtime` / kubebuilder** | scaffolds the CRD, cache, queue, leader election, metrics; you write `Reconcile()` only | **the default** |
| Operator SDK | kubebuilder + packaging/lifecycle conventions | OLM/marketplace ecosystems |
| Kyverno / plain CronJob / a script | no controller at all | *most cases — see below* |

## When NOT to write an operator

The honest section. An operator is a distributed-systems component you're
signing up to run forever: versioned APIs (7.5's storage-version migrations),
upgrade paths, RBAC, on-call. Reach for one only when you have **ongoing
reconciliation of custom state** — something must watch and converge
*continuously* (databases with failover, per-tenant provisioning).

If the need is "run a thing on a schedule" — a CronJob (2.5). "Enforce a
policy" — Kyverno/Gatekeeper (M6/7.7's admission seat). "Deploy from git" —
Argo/Flux, which is someone else's operator (3.6). "One-time setup" — a Job.
The strongest internals knowledge in this module is knowing the loop well
enough to recognize when you don't need to build one.

## Try the 20% version

kubebuilder's quickstart gets `BackupSchedule` reconciling in an afternoon on
a kind cluster — the scaffold generates everything around the `Reconcile()`
you can now write from memory. Do it once; the whole platform (Deployments,
StatefulSets, cert-manager, everything in 7.1's controller chain) stops being
machinery and becomes *peers*: the same 60 lines, tiered.

*No check — study, then advance.*
