> **Capstone incident file (guided study).** No lab — this failure is a
> multi-system cascade that can't be honestly reproduced on kind. Instead you'll
> trace it with everything you now know. Every link in the chain is a mechanism
> you've already fixed in isolation; the lesson is how they *compose*.
>
> Sources:
> [Monzo status thread (Oct 2017)](https://community.monzo.com/t/resolved-current-account-payments-may-fail-major-outage-27-10-2017/26296/95) ·
> [Anatomy of a Production Kubernetes Outage — KubeCon talk](https://www.youtube.com/watch?v=OUYTNywPk-s)

## Why this is the capstone

Monzo is a bank. On 27 Oct 2017, current-account payments failed for ~1.5 hours —
real people couldn't pay for things. The trigger was mundane. The outage was not.
This is what "know Kubernetes in and out" is *for*: not fixing one broken object,
but seeing how a small change propagates through the platform's layers until a
bank stops.

## The chain (each link is a lesson you've done)

**Link 1 — a routine change meets etcd.**
Monzo was scaling and made an infrastructure change involving their **etcd**
cluster (the store from Module 7.3). etcd is the source of truth *and* the thing
Kubernetes and their service mesh both watched. A change here doesn't stay here —
it ripples to everything watching.

**Link 2 — the mesh (linkerd) gets a bad view.**
Their service mesh consumed Kubernetes service discovery. During the etcd
disruption, the information the mesh received about *which pods back which
service* became wrong — effectively the **empty-endpoints** condition from
Modules 1 and 4, but arriving through the mesh's discovery path rather than a
selector typo. Services that were actually healthy started looking like they had
**no endpoints**.

**Link 3 — empty endpoints become client crashes.**
Here's the vicious turn. A client service, handed an empty endpoint list for a
dependency, hit a code path that didn't expect it — a **null-pointer exception**.
The clients didn't degrade gracefully; they *crashed*. Now you don't just have a
discovery blip, you have crashing pods (Module 1's CrashLoopBackOff), which
churn endpoints further, which feeds more bad discovery data back into the mesh.

**Link 4 — the feedback loop.**
Crashing clients → more endpoint churn → more services appearing endpoint-less →
more clients crashing. A **reconciliation storm** with positive feedback: the
system's own healing machinery (restart, re-register, re-discover) amplified the
fault instead of damping it. This is the difference between an incident and an
*outage* — whether the loops converge or diverge.

**Link 5 — blast radius = the business.**
Because payments sat downstream of the affected services, the cascade surfaced
as "current-account payments may fail." Five infrastructure mechanisms, and the
symptom is a customer at a till.

## Trace it yourself (concept checks)

You have the cluster from earlier lessons; reason with these:

```sh
# Link 2/3: what a client sees when a dependency has no endpoints —
kubectl -n kubelings get endpoints            # empty ENDPOINTS = a blackhole
# Link 4: how churn shows up —
kubectl -n kubelings get events --field-selector reason=SuccessfulCreate,reason=Killing
```

- **Where would *you* have cut the loop?** (graceful handling of empty
  endpoints? circuit breakers? a mesh that fails static instead of empty?)
- **Which single lesson, applied, breaks the chain earliest?** (Link 3: a client
  that treats "no endpoints" as "retry/– degrade", not "crash", stops the
  feedback loop dead.)

## The lessons Monzo drew (and the industry adopted)

- **Graceful degradation over crashing.** Empty/failed dependency = handled
  error, backoff, circuit-break — never an unhandled exception. A crash converts
  *your* problem into *everyone downstream's* problem.
- **Understand shared-fate dependencies.** etcd underpinned both Kubernetes and
  the mesh — one disruption, two systems misbehaving in correlated ways. Map
  what shares fate *before* the incident (echoes Zalando's monitoring-needs-DNS
  trap, Module 4).
- **Beware self-amplifying recovery.** Restart/re-register/retry are healing at
  low scale and gasoline at high scale. Add jitter, backoff, circuit breakers,
  and rate limits so loops converge.
- **Blast-radius thinking.** Put failure domains between "infra change" and
  "payments." Bulkheads, not one connected mass.

## How this course adds up

Look back at what each link *was*:

| Link | Mechanism | Where you learned it |
|---|---|---|
| 1 | etcd as shared source of truth | M7 Internals |
| 2 | service discovery / endpoints | M1, M4 |
| 3 | empty endpoints, crashloops | M1 |
| 4 | reconciliation & feedback loops | M7 |
| 5 | blast radius / dependencies | M5, M8 |

None of it was new. **Mastery is seeing the whole chain while it's happening** —
which link to cut, which loop is diverging, where the next blast will land. That
is knowing Kubernetes as a platform, in and out. Take the final boss next.

*No check — study, then advance.*
