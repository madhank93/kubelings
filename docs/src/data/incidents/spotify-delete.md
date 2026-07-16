---
title: "Spotify: deleted every production cluster — twice — with no user impact"
description: "[REAL] 2018–2019 — during Spotify's GKE migration, production Kubernetes clusters were accidentally deleted, twice. Users barely noticed. The war story where the interesting part is why nothing happened."
---

> **[REAL] incident** — cited from David Xia's KubeCon + CloudNativeCon
> Europe 2019 keynote:
> [How Spotify Accidentally Deleted All its Kube Clusters with No User Impact](https://www.youtube.com/watch?v=ix0Tw8uinWs).
> **Related Kubelings lessons:** `incident-spotify-delete` (M9),
> `etcd-backup-restore` (M7), `gitops-argocd` (M10).

## Situation

2017: Spotify plans a migration of hundreds of teams, thousands of services,
and tens of thousands of hosts to Google Kubernetes Engine. By late 2018,
50 teams and hundreds of services — including critical ones — run on
multiple production clusters, while the legacy Helios-based infrastructure
still runs alongside.

Then, during the migration, David Xia's team **accidentally deleted most of
their production Kubernetes clusters. Twice.**

## Blast radius

- Production clusters destroyed — the compute layer under hundreds of
  services, gone.
- **Little to no user impact.** That's the headline, and it wasn't luck.
- Recovery cost: hours of cluster re-creation and workload re-scheduling per
  event, plus the postmortem work that reshaped how Spotify manages clusters.

## Why users didn't notice

The migration strategy *was* the safety net:

1. **Parallel infrastructure** — services still existed on the legacy
   platform during migration; traffic could fall back rather than fall over.
2. **Multiple clusters, multiple regions** — no single cluster was the only
   home of anything; capacity elsewhere absorbed the loss.
3. **Stateless, declaratively-deployed services** — what defines a service
   lived outside the cluster, so a cluster was cattle: expensive to lose,
   possible to rebuild.

The cluster turned out to be **disposable infrastructure**; what mattered
was that the *definitions* (services, configs, deployment pipelines) lived
somewhere durable.

## Fix & prevention

From the postmortems, Spotify rebuilt cluster operations around removing the
human from the loop:

- **Clusters declaratively defined in code with Terraform** — creation and
  deletion became reviewed, versioned changes instead of imperative commands
  against production. (An interactive `gcloud`-style deletion can't be code
  reviewed; a Terraform plan can.)
- **Backup and restore with Ark** (now Velero) — cluster state became
  restorable, and restores became a practiced operation instead of a
  hypothesis.
- **Run many more clusters** — smaller blast radius per cluster, and
  operating "many" forces the automation that operating "a few" lets you
  postpone. The same conclusion Target's cascade reached from the opposite
  direction.

## What it teaches

| Concept | Kubelings module |
|---|---|
| clusters as cattle: state lives outside, restore is practiced | M7 — `etcd-backup-restore` |
| declarative infrastructure as the blast-radius control | M10 — `gitops-argocd` |
| migration-era redundancy is a feature, not waste | M9 — war stories |
| blameless postmortems turn disasters into platform upgrades | M9 — war stories |
