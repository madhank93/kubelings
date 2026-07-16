> **Capstone incident file (guided study).** No lab — the failure needs
> thousands-of-nodes scale to exist at all. Read it with Module 7 fresh: this
> whole outage is the control-plane/data-plane boundary you toured, breaking in
> public.
>
> Source:
> [OpenAI status — API, ChatGPT & Sora facing issues (11 Dec 2024)](https://status.openai.com/incidents/ctrsv3lwd797)

## What happened

On 11 December 2024, ChatGPT, the API, and Sora went down for roughly four
hours. Nobody shipped a bad model or a bad app build. What shipped was a
**telemetry service** — a DaemonSet-style agent meant to *improve*
observability of the control plane — deployed **to every cluster in the fleet
within a short window**.

The agent's API usage was expensive in a way that scaled with cluster size:
across thousands of nodes, its Kubernetes API requests amounted to a sustained
overload of every **API server** simultaneously. It had been tested — but not
at the size of the largest production clusters, where per-node cost × node
count crossed the line.

## The chain

**Link 1 — the control plane drowns.** kube-apiserver (Module 7's front door)
saturates. Every controller, every kubectl, every watch: degraded or dead.
Note what this is: the **quota lesson's aggregate-cap logic, missing**. Each
agent's requests were individually reasonable; the *sum* was not, and nothing
enforced a ceiling on the sum.

**Link 2 — the data plane should have shrugged. It didn't.** Your Module 7
mental model: workloads keep running when the control plane dies — kubelets
and containers don't need the API server minute-to-minute. That held! Pods
stayed up. But **service discovery ran through DNS, and DNS records were
resolved from cluster state**. With the control plane down, DNS answers
couldn't be refreshed. The data plane had a hidden control-plane dependency —
Zalando's trap (M4) and Monzo's shared-fate lesson, in a third costume.

**Link 3 — DNS caching turns safety into a fuse.** DNS caches (ndots and
friends, Module 4) kept serving stale-but-valid answers for ~20 minutes. Two
brutal effects: the rollout **looked healthy** long enough to reach the whole
fleet before symptoms appeared — defeating any "watch the canary" instinct —
and then the caches expired *fleet-wide on a timer*, detonating everywhere at
once. A cache is a time-delayed dependency: it doesn't remove the coupling,
it postpones your discovery of it.

**Link 4 — the lockout.** The fix was obvious in minutes: delete the telemetry
deployment. But deleting a deployment requires… the Kubernetes API — the very
thing that was down. The operators were **locked out by their own incident**.
Recovery meant prying the door open: shrinking clusters to cut API load,
blocking non-admin API traffic at the network level, and scaling API servers
up — all to win back enough control-plane headroom to issue one delete.

## Concept checks

- Sketch the dependency edge that should not have existed. Which direction
  does it point, and what breaks when you cut it? (Data plane → control plane
  for *runtime* request serving. Cut it: pods serve traffic with no API server
  at all, discovery data is pushed/pinned rather than resolved live.)
- Your canary rollout of a node agent looks healthy for 30 minutes. Given Link
  3, what's the *minimum* honest bake time? (Longer than every cache TTL and
  periodic-resync interval the agent's blast radius touches.)
- Why did "tested in staging" fail here when it usually works? (The failure was
  a *scale product* — per-node cost × nodes. Staging validates correctness;
  only the largest cluster validates aggregate load.)

## What the industry took from it

- **Stagger fleet rollouts, gate on time not just health.** Healthy-looking is
  not healthy when caches delay symptoms; waves with real bake time bound the
  blast radius to wave one.
- **Break-glass access.** An emergency path to the control plane that doesn't
  compete with normal API traffic (and is exercised regularly) turns a
  four-hour lockout into a ten-minute rollback.
- **Decouple the data plane.** Runtime request-serving must survive control
  plane loss: cached/pushed endpoints, discovery that fails static (compare
  the webhook lesson's `failurePolicy`, M6 — same "fail safe, not fail
  coupled" principle).
- **Protect the API server from its tenants.** API Priority and Fairness,
  per-client rate limits, and load-testing agents against the *biggest*
  cluster — because the API server is the one shared service every fix flows
  through.

*No check — study, then advance.*
