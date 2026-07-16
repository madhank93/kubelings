> **Capstone incident file (guided study).** No lab — the fault lives in a
> years-old snowflake cluster's Calico BGP config meeting a Kubernetes upgrade;
> what's reproducible here is the *reasoning*. Read the postmortem itself
> afterwards — it's one of the most honest ever published.
>
> Source:
> [RedditEng — "You Broke Reddit: The Pi-Day Outage"](https://www.reddit.com/r/RedditEng/comments/11xx5o0/you_broke_reddit_the_piday_outage/)

## What happened

14 March 2023. Reddit's infra team starts a routine, well-practiced Kubernetes
upgrade (1.23 → 1.24) on their oldest, largest cluster — the one running the
site's legacy core. Minutes in, the whole cluster goes dark. Pods can't reach
pods. The site is down, and stays degraded for about **five hours** — until the
team does the thing nobody ever wants to do: **restore the cluster from
backup**.

## The chain

**Link 1 — a deprecation lands.** Kubernetes 1.24 completed a long-announced
rename: the control-plane node label `node-role.kubernetes.io/master` was
removed in favor of `node-role.kubernetes.io/control-plane`. Upgraded nodes
simply stopped carrying the old label. Labels and selectors — lesson 1.4, the
selector mismatch — except this time *the platform* changed the label out from
under the selector.

**Link 2 — the selector goes empty.** This cluster's Calico CNI ran BGP
**route reflectors** — the nodes that redistribute pod-network routes to
everyone else — chosen by a node selector that matched on the **old `master`
label**. Label gone ⇒ selector matches zero nodes ⇒ no route reflectors ⇒ BGP
mesh collapses ⇒ **no node knows how to route to pods anymore**. Same organ
Datadog lost (M8's incident file: CNI = routes on hosts), different killer:
there it was the OS deleting routes; here the route *distribution* died of an
empty selector.

**Link 3 — the snowflake tax.** Why did no one know? The cluster predated
Reddit's standardized cluster tooling; its Calico setup was hand-configured
years earlier and *unique to it* — newer clusters didn't have the landmine.
The engineers who built it had moved on. Institutional memory is a dependency
with no healthcheck: every hand-crafted difference between this cluster and
your standard build is a page waiting for the person who doesn't know it's
there.

**Link 4 — recovery is its own incident.** Roll back? Kubernetes downgrades
are officially unsupported; the path was a **full restore from etcd backup**
(M7.3 — etcd is the cluster) — under pressure, on a cluster whose restore
procedure had never been exercised at this scale, with TLS and component
config mismatches fighting them down the stretch. Untested restores aren't
recovery plans; they're recovery *hypotheses*. Total: ~314 minutes. On Pi Day.

## Concept checks

- The upgrade *itself* succeeded — API server, kubelets, etcd all fine. What
  monitoring signal actually mattered, and at which layer? (Pod-to-pod
  dataplane reachability — a synthetic that would've flatlined instantly.
  Node Ready and control-plane health stayed green: kubelet heartbeats don't
  traverse the pod network.)
- Where in *your* cluster do selectors reference platform-owned labels?
  `kubectl get <thing> -o yaml | grep -B2 -A4 nodeSelector` on CNI, ingress,
  and monitoring stacks — would a label rename strand any of them?
- Reddit's team rehearsed this upgrade on other clusters and it went fine. Why
  didn't that de-risk *this* one? (Rehearsal covers what's *shared*. Nothing
  rehearses a snowflake but itself — which is the argument for making it not
  be one.)

## What the industry took from it

- **Read the deprecation notes like a diff against your own configs.** The
  `master`→`control-plane` rename was announced releases in advance; a grep of
  cluster addon configs for the dying label would have found the route
  reflectors' selector in seconds.
- **Kill snowflakes before they kill you.** One templated, versioned cluster
  build (the newer Reddit clusters were fine!); config drift from the standard
  is a bug, not a quirk.
- **Practice the restore, not just the backup.** An etcd restore you've never
  run is Link 4 waiting to happen. Game-day it on a real (non-prod) cluster,
  including certs and addon config.
- **Upgrade the oldest cluster first in staging form, not last in prod form** —
  clone its quirks somewhere disposable if you must keep it.

*No check — study, then advance.*
