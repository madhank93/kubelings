> **Capstone incident file (guided study).** No lab — you can't (and
> shouldn't) replay deleting production clusters; what's reproducible is the
> *architecture that made it survivable*. Watch the keynote — it's twenty
> minutes and worth every one.
>
> Source:
> [David Xia (Spotify), KubeCon EU 2019 keynote — "How Spotify Accidentally Deleted All its Kube Clusters with No User Impact"](https://www.youtube.com/watch?v=ix0Tw8uinWs)
>

## What happened

2017: Spotify plans a migration of hundreds of teams, thousands of services
and tens of thousands of hosts to GKE. By late 2018, 50 teams and hundreds
of services — including critical ones — run on multiple production
clusters, with the legacy platform still alongside.

During that migration, the infrastructure team **accidentally deleted most
of their production Kubernetes clusters. Twice.** Users barely noticed.

This is the inverted war story: every other incident file in this module
asks "why did such a small fault cause so much damage?" This one asks
**"why did such a huge fault cause so little?"**

## The chain — of defenses, not failures

**Defense 1 — parallel infrastructure.** Mid-migration, services still
existed on the legacy platform. The migration's "waste" — running two
platforms at once — was the fallback path.

**Defense 2 — many clusters, no snowflakes.** No cluster was the only home
of anything. Contrast Reddit's Pi-Day (M9.3): their oldest cluster was
unique, so losing it was unsurvivable-by-design. Spotify's were
interchangeable, so losing them was an inconvenience.

**Defense 3 — state outside the cluster.** Service definitions, configs and
deployment pipelines lived in durable systems *outside* the clusters being
deleted. A cluster was cattle: rebuilding it was work, not loss. (M7.3 —
etcd is the cluster; so if the cluster must be disposable, the *source of
truth* can't live only in etcd.)

## Concept checks

- Delete your production cluster in your head right now. What's actually
  gone? Separate "state that exists only in etcd" from "state derivable from
  git/CI/registries" — the first list is your real blast radius.
- Twice. The second deletion came from *automation being introduced* to
  prevent the first. What does that tell you about testing infrastructure
  code against production state? (Plans/dry-runs are the code review of
  infrastructure change.)
- Why does "run many more clusters" *reduce* operational risk when it
  multiplies the number of things to operate? (It forces automation and
  uniformity — the alternative is a few hand-tended snowflakes.)

## What the industry took from it

Spotify's post-incident changes, from the keynote:

- **Clusters declaratively defined in code (Terraform)** — cluster
  create/delete became reviewed, versioned changes, not interactive commands
  against production.
- **Backup and restore with Ark (now Velero)** — restore as a practiced
  operation, not a hypothesis. Reddit's Pi-Day (M9.3) shows the cost of the
  untested alternative.
- **Run many more clusters** — smaller blast radius each; Target's cascade
  (M9.5) reached "smaller clusters, more of them" from the opposite
  direction: theirs was too big to survive, Spotify's were numerous enough
  to lose.
- **Blameless postmortems** — the engineer who deleted production keynoted
  KubeCon about it. That's the culture that turns disasters into platform
  upgrades. Module 10 builds the GitOps tooling that makes this posture
  practical.

*No check — study, then advance. The final boss awaits.*
