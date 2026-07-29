---
title: "Skyscanner: a couple of characters pruned 478 services worldwide"
description: "[REAL] 2021 — missing template delimiters in a root config file rendered to nothing; GitOps reconciled the emptiness by deleting every service in every region. Runnable Kubelings replay."
---

> **[REAL] incident** — cited from Skyscanner Engineering's public write-up:
> [How a couple of characters brought down our site](https://medium.com/@SkyscannerEng/how-a-couple-of-characters-brought-down-our-site-356ccaf1fbc3).
> **Runnable replay:** Kubelings lesson `incident-gitops-prune`
> (Module 10 — Platform Engineering).

## Situation

24 August 2021. A change intended as a no-op edited a configuration file at the
root of Skyscanner's "cells" tree — the file that distributes regional
configuration to every cell. The edit dropped the template's `{{ }}` delimiters.

## Blast radius

- **478 services deleted** across all namespaces, availability zones and regions.
- skyscanner.net and the mobile apps down globally for ~4.5 hours.
- Full restoration of every region and service took a further two days.
- The Cells Architecture — built to contain *regional* failure — could not
  contain a *central configuration* failure.

## Root cause chain

1. **A high-blast-radius file changed.** Rarely touched, top of the tree, rolled
   out globally and immediately by design.
2. **The renderer failed soft.** Without delimiters the templating produced
   nothing usable — no error, just an empty result.
3. **The GitOps agent obeyed.** Argo CD compared desired state (now: no valid
   namespaces, no services) with live state and reconciled the difference the
   only way it knows: deletion.
4. **Nothing distinguished "nothing rendered" from "delete everything."** That
   distinction is not automatic; it is a guard someone has to write.

## Fix & prevention (Skyscanner's own conclusions)

- **Treat configuration as code:** lint, schema-check and render-test it in CI.
  "Renders to zero resources" is a test, and a cheap one.
- **Fail closed on empty desired state.** Argo CD ships the upstream form of this
  guard — `automated.prune: false` where you can live without pruning, and
  `allowEmpty: false` so an Application that renders nothing refuses to sync.
- **Avoid global-and-immediate rollouts.** Regional config should move region by
  region with a bake time.
- **Test the restore path**, thoroughly and often — recovery here took two orders
  of magnitude longer than the failure, and the gap was procedural.
- **Write runbooks for stressed engineers**, not for the person who wrote them.

## What it teaches

| Concept | Kubelings module |
|---|---|
| GitOps reconciliation & prune semantics | M10 — `incident-gitops-prune` (runnable) |
| Argo CD sync policy & app-of-apps blast radius | M10 — `gitops-argocd`, `gitops-argocd-appofapps` |
| Declarative config as a deletion instruction | M9 — `incident-spotify-delete` (same shape via Terraform) |
| Templating/overlay correctness | M3 — `kustomize-overlays`, `helm-releases` |
