---
title: "Heroku: the auto-update that severed the fleet"
description: "[REAL] Jun 2025 — an automated OS update restarted host networking fleet-wide; a legacy script only applied routes on first boot, so restarts silently severed dyno networking. ~24h platform outage — the Datadog 2023 failure mode, recurring."
---

> **[REAL] incident** — cited from Heroku's official summary:
> [Summary of Heroku June 10 Outage](https://www.heroku.com/blog/summary-of-june-10-outage/)
> · [status page incident](https://status.heroku.com/incidents/2822).
> **Related Kubelings lessons:** `incident-datadog-cilium` (M8 — the same
> failure shape, two years earlier), `upgrade-runbook` (M8).

## Situation

06:00 UTC, Tuesday 10 June 2025 — Heroku customers start seeing intermittent
logins, failing applications, and dynos that can't make outbound requests.
The disruption runs **up to 24 hours** for many customers. Not a security
incident; no customer data lost. The trigger: an **unintended automated
operating-system update** ran across production infrastructure.

## Blast radius

- The majority of dynos in Private Spaces unable to make outbound HTTP
  requests; widespread application failures and login errors.
- **Heroku's own tools and the Status Page ran on the same affected
  infrastructure** — as customer apps failed, so did Heroku's ability to
  respond and communicate. Status updates eventually went out through a
  workaround on X at 13:58, nearly eight hours in.
- Long-tail cleanup into the next morning: queued status emails, Heroku
  Connect sync backlogs, release-phase backlogs. Declared resolved 05:50 UTC
  on 11 June.

## Root cause chain

Heroku's postmortem names three compounding issues:

1. **A control issue** — an automated OS update ran on production hosts when
   it should have been disabled. The update restarted the host's networking
   services.
2. **A resilience issue** — the networking service depended on a **legacy
   script that only applied routing rules on initial boot**. On restart, the
   routes were never reapplied — severing outbound network connectivity for
   every dyno on the host. Nothing crashed; the routes were just *gone*.
3. **A design issue** — internal tooling and the public status page shared
   the failing infrastructure, so response and communication were impaired
   exactly when they were needed most.

Root-causing took most of a day precisely because nothing was "down":
engineers compared healthy and unhealthy hosts to spot the **missing network
routes** (11:54), discovered the unexpected network service restart (13:11),
and only then pinned the automated package upgrade as the trigger (13:42).

## Fix & prevention

- **Stop the trigger**: the upstream vendor's auto-update token was
  invalidated (confirmed 17:30, completed 19:18) so no further hosts could be
  touched; then a fleetwide dyno recycle stabilized remaining services.
- **Immutability controls** so automated processes cannot make unplanned
  changes to production — the corrective action Datadog's 2023 postmortem
  also landed on.
- **Auto-update channels disabled / audited base images** — an update you
  didn't schedule is an unreviewed production change.
- **Don't host your incident response on the infrastructure it monitors.**

## What it teaches

| Concept | Kubelings module |
|---|---|
| the OS under the cluster can delete what your CNI installed | M8 — `incident-datadog-cilium` |
| routes applied only at boot = state that can't survive a restart | M4 — `kube-proxy-dataplane` |
| change discipline: staged, observed, reversible | M8 — `upgrade-runbook` |

The same root shape has now taken down two major platforms publicly — Datadog
in March 2023, Heroku in June 2025. Postmortems from other companies are only
valuable if they change *your* fleet.
