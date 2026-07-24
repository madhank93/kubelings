---
kind: unit
title: "Incident file — the OS under the cluster (Datadog, 2023)"
name: incident-datadog-cilium-unit
---


> **Incident file (guided study).** No lab — you can't honestly reproduce
> "an OS security update breaks the CNI on tens of thousands of nodes" on kind.
> You *can* now read this postmortem fluently: every mechanism in it has a
> lesson behind it.
>
> Source:
> [Datadog — 2023-03-08 infrastructure connectivity issue](https://www.datadoghq.com/blog/2023-03-08-multiregion-infrastructure-connectivity-issue/)

## What happened

On 8 March 2023, starting around 06:00 UTC, Datadog — a company whose product
*is* observability — went dark for over a day. Not one cluster: **tens of
thousands of nodes, across five regions, on multiple cloud providers, within
about an hour of each other.**

The trigger was as mundane as triggers get: a **security update to systemd**,
delivered by the OS's automatic update mechanism on Ubuntu nodes. Applying the
update restarted `systemd-networkd`. And on restart, systemd-networkd removed
routes it didn't own — including the routes **Cilium**, the CNI plugin, had
installed to carry pod traffic.

Route table wiped → pod networking gone → node effectively off the network.

## Why the blast radius was total

Each piece is something you've already touched; the outage is their
composition:

**1. The layer below Kubernetes.** The CNI (Module 4) isn't magic — on every
node it's ultimately *routes and interfaces in the host's network stack*, owned
by the host OS. Kubernetes never saw a misconfiguration because nothing in
Kubernetes changed: no deploy, no manifest, no kubectl. The change arrived via
`apt`. Your cluster's real config surface includes the OS image.

**2. Correlated execution = no failure domains.** Automatic updates ran in the
same overnight window fleet-wide, so a per-node fault became an
*everything-at-once* fault. Regions, clouds, clusters — all irrelevant as
isolation boundaries, because the update channel cut across all of them.
Compare Zalando's DNS incident (Module 4) and Monzo's shared etcd (Module 9):
same lesson, different shared dependency. **Blast radius follows the thing
everything has in common.**

**3. The NotReady cascade at fleet scale.** From the control plane's view this
was the node lifecycle you drilled two lessons ago, times ten thousand: kubelet
heartbeats stop, nodes flip NotReady, `not-ready`/`unreachable` taints land,
and after `tolerationSeconds` the taint manager starts **mass-evicting pods
that have no healthy node to land on**. The platform's per-node healing
machinery, pointed at a fleet-wide fault, amplified churn instead of fixing
anything — the same divergent-feedback shape as Monzo's cascade.

**4. Recovery is a capacity problem.** Fixing the route tables wasn't the slow
part. The slow part: rebooting/replacing compute at fleet scale while every
workload tries to reschedule at once — control planes and autoscalers under
their worst-ever load, quotas and image pulls (Module 1) throttling the
stampede. Restoring a platform is itself a scheduling problem (Module 5), and
it's why the outage lasted *days* in the tail, not the minutes the fix took.

**5. The observer went down with the observed.** Datadog's own status page and
comms lagged the outage — their tooling ran on the affected infrastructure.
Shared-fate again, one level up: *monitoring must not depend on what it
monitors.* (Zalando's monitoring-needs-DNS trap, at company scale.)

## Concept checks

- A node's Cilium routes vanish but the kubelet process is fine. Walk the
  timeline: how long until NotReady? What taint appears? When do evictions
  start? (You know the numbers: ~40s grace, 300s toleration.)
- Why did "five regions on three clouds" provide zero protection here? What
  *would* have? (Staggered update rings — the thing that cuts across regions
  must itself be staggered.)
- Your fleet auto-applies security patches (it should — the alternative is
  unpatched fleets). What's the difference between that and what bit Datadog?
  (Not *whether* but *how*: canary ring → bake time → waves, with a node-level
  health gate between waves.)

## What to steal for your own fleet

- **Ring-deploy the OS layer** like you ring-deploy apps: canary nodes first,
  automated verification (CNI healthy? routes present? node Ready?), then
  waves. An update channel with no rings is a global simultaneous deploy you
  didn't schedule.
- **Pin and bake node images.** Immutable, versioned node images (rebuilt to
  pick up CVEs, rolled like any release) turn "apt did something at 6am" into
  a reviewable, rollback-able change.
- **Know your CNI's failure mode.** Who owns the routes on your nodes? What
  else touches them? `systemd-networkd` vs Cilium was a *known* interaction
  with a config mitigation — the kind of thing a periodic "what owns what on a
  node" review catches.
- **Run monitoring out-of-band.** Status pages, paging, and the dashboards you
  need *during* the incident live on infrastructure that fails separately.

## History rhymed

Two years later it happened again to someone else:
[Heroku's June 10, 2025 outage](https://www.heroku.com/blog/summary-of-june-10-outage/)
— an unattended system update made unplanned changes across production and
flushed network routes fleet-wide, ~24 hours of platform downtime. Their
corrective actions read like this lesson's checklist: immutability controls
so automated processes can't mutate production, disabling the auto-upgrade
channel, auditing base images. Postmortems from other companies are only
valuable if they change *your* fleet — this failure mode has now taken down
two major platforms, publicly, with the same root shape.

*No check — study, then advance. Module 9 shows what these cascades look like
from inside the incident channel.*
