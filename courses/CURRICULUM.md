# Kubelings curriculum backlog

Authoring roadmap. `[x]` = shipped. Every lesson = `index.md` (init/verify tasks)
+ `unit-1.md` (situation → task → hint → check → solution). **Slug = lesson dir
basename minus the `N.` prefix** and must be globally unique (progress is keyed
by slug in `.labctl/progress.tsv`). Incident-replay lessons use the `incident-`
dir prefix, pattern drills use `pattern-`; the TUI derives the type badge
(replay / drill / read / lab) from that prefix + whether tasks exist. Replay
lessons carry a verified `source:` URL (see
`docs/src/content/docs/reference/incident-library.md`).

## Module 1 — Foundations
- [x] 1.welcome
- [x] 2.crashloop-triage · [x] 3.expose-web · [x] 4.selector-mismatch
- [x] 5.namespace-basics · [x] 6.imagepull-backoff · [x] 7.kubectl-detective

## Module 2 — Workloads
- [x] 1–7 (rolling-update, daemonset, statefulset, jobs, cronjobs, hpa, oomkill)
- [x] 8.liveness-vs-readiness · [x] 9.startup-probe · [x] 10.init-containers
- [x] 11.pdb-blocks-drain · [x] 12.qos-classes · [x] 13.blue-green-canary
- [x] 14.incident-cpu-throttling [REAL Omio/Buffer]
- [x] 15.ephemeral-containers · [x] 16.multi-container-patterns
- [x] 17.pattern-readiness-flap [DRILL] · [x] 18.pattern-zombie-cronjobs [DRILL]
- [x] 19.pattern-rolling-update-deadlock [DRILL]
- [ ] 20.vpa (recommender-only manifests) · [ ] 21.keda-autoscaling (cpu/cron
      trigger — no Prometheus dependency)

## Module 3 — Config & Storage
- [x] 1.configmap-wiring · [x] 2.secret-not-mounted · [x] 3.pvc-pending
- [x] 4.access-modes · [x] 5.pattern-pvc-terminating [DRILL]
- [x] 6.kustomize-overlays
- [x] 7.helm-releases (init installs a pinned helm binary — unblocks the old
      "helm not guaranteed" note)
- [x] 8.pattern-ghost-endpoints [DRILL] · [x] 9.pattern-secret-not-reloaded [DRILL]
- [x] 10.pattern-namespace-terminating [DRILL]

## Module 4 — Networking
- [x] 1.incident-dns-ndots [REAL Zalando] · [x] 2.networkpolicy-blackhole
- [x] 3.broken-targetport · [x] 4.nodeport-vs-clusterip
- [x] 5.incident-conntrack [REAL loveholidays/Preply, reading]
- [x] 6.incident-graceful-shutdown [REAL Ravelin] · [x] 7.ingress-wiring
- [x] 8.gateway-api (init fetches pinned CRDs — needs internet from cplane-01)
- [x] 9.kube-proxy-dataplane [reading: iptables/IPVS/nftables mechanics]
- [ ] 10.cni-basics [reading: conflist anatomy, kubelet↔CNI, crictl triage —
      also covers the old M7 "CNI hands-on" backlog; NetworkPolicy debugging
      stays in 2.networkpolicy-blackhole]
- [ ] 11.kubeconfig-contexts

## Module 5 — Scheduling & Placement
- [x] 1.incident-same-node [REAL Moonlight] · [x] 2.taints-tolerations
- [x] 3.topology-spread · [x] 4.incident-priority-preemption [REAL Grafana]
- [x] 5.pattern-noisy-neighbor [DRILL]

## Module 6 — Security
- [x] 1.rbac-least-privilege · [x] 2.incident-cryptominer [REAL JW Player]
- [x] 3.incident-webhook-outage [REAL Jetstack] · [x] 4.pod-security-baseline
- [x] 5.serviceaccount-tokens · [x] 6.container-hardening · [x] 7.cis-kube-bench
- [x] 8.control-plane-hardening [reading: encryption at rest, audit, supply chain, runtime]
- [x] 9.egress-lockdown · [x] 10.image-digests
- [ ] 11.opa-gatekeeper · [ ] 12.kyverno-policies
- [ ] 13.image-scanning-pipeline (init installs pinned trivy)
- [ ] 14.sbom-cosign (init installs pinned cosign; client-side verify lab)
- [ ] 15.apparmor-seccomp (hands-on seccomp RuntimeDefault; AppArmor as
      reading section — host-level per confinement policy)
- [ ] 16.encryption-at-rest [reading: runbook — deep dive from
      8.control-plane-hardening §1]
- [ ] 17.audit-policy [reading: runbook — deep dive from §2]
- [ ] 18.falco-runtime-detection [reading: runbook — deep dive from §5]

## Module 7 — Internals
- [x] 1.reconcile-loop · [x] 2.scheduler-nodename · [x] 3.etcd-truth
- [x] 4.control-plane-tour [reading: api-flow, kubelet/CRI, leader election]
- [x] 5.crd-operators · [x] 6.etcd-backup-restore [reading: snapshot/restore runbook]
- [x] 7.admission-mutations · [x] 8.watch-informers [reading, incl. APF]
- [x] 9.build-an-operator [reading: controller walkthrough capstone]
- [ ] 10.kubeadm-bootstrap [reading: init→join runbook — cannot run inside
      k8s-omni; practice on iximiuz multi-VM playground]
- [ ] 11.ha-control-plane [reading: stacked vs external etcd, kind 3-CP
      concept config]
- [ ] 12.cert-rotation [reading: kubeadm certs check-expiration/renew runbook]
- [ ] backlog: CSI hands-on (CNI reading now covered by M4 10.cni-basics)

## Module 8 — Observability & SRE
- [x] 1.events-forensics · [x] 2.incident-node-oom [REAL Blue Matador]
- [x] 3.quota-exhausted · [x] 4.node-notready · [x] 5.pattern-disk-pressure [DRILL]
- [x] 6.incident-datadog-cilium [REAL Datadog, reading]
- [x] 7.upgrade-runbook [reading: version skew, kubeadm, drain/uncordon]
- [ ] 8.node-maintenance (cordon/drain/uncordon full cycle — contrasts with
      M2 pdb-blocks-drain where drain was the *problem*)
- [ ] 9.slo-errorbudget (prometheus-operator bundle manifests, NOT Helm;
      merged with the platform-SLO-dashboards idea — one lab, both bugs)
- [ ] 10.otel-collector-pipeline (collector Deployment + Jaeger all-in-one)

## Module 9 — War Stories (capstone)
- [x] 1.incident-monzo-cascade [REAL Monzo, reading]
- [x] 2.incident-openai-cascade [REAL OpenAI, reading]
- [x] 3.incident-reddit-piday [REAL Reddit, reading]
- [x] 4.incident-black-friday [REAL Algolia, reading]
- [x] 5.incident-target-cascade [REAL Target, reading]
- [x] 6.incident-spotify-delete [REAL Spotify, reading]
- [x] 7.final-boss [multi-fault, no hints] (renumbered from 5. on 2026-07-13)

## Module 10 — Platform Engineering (NEW)
- [ ] 1.gitops-argocd (pinned install.yaml; verify via kubectl jsonpath — no
      argocd CLI assumption)
- [ ] 2.gitops-argocd-appofapps · [ ] 3.gitops-flux2 (pinned install.yaml,
      not `flux bootstrap`)
- [ ] 4.multi-tenancy-capsule · [ ] 5.cluster-api-intro [reading]
- [ ] 6.crossplane-compositions

## Incident library (docs)
- [x] Index seeded with 40+ verified [REAL] rows + 14 [PATTERN] stubs; runnable/reading
      lessons cross-linked
- [x] Case studies: Zalando DNS, [PATTERN] PVC Terminating, CircleCI version
      skew, Heroku host-update, Target cascade, Spotify delete, + more
- [x] Dropped (2026-07-13, verify-first policy): Weaveworks GitOps-divergence
      row — weave.works domain dead, no Wayback snapshot
- [ ] Add a case-study page per remaining [REAL] row as bandwidth allows
- [ ] Grow toward ~50 [REAL] / ~30 [PATTERN]; verify every URL before adding

## Cert coverage notes (CKA/CKAD/CKS)

- Host-level exam topics (kubeadm upgrade exec, etcdctl restore exec, AppArmor
  profiles, Falco install) are **readings** with full runbooks — lesson scripts
  are kubectl-only by design (see the security confinement commit).
- NetworkPolicy labs verify objects; kind's default kindnet does NOT enforce
  them (called out in the units) — enforcement testing happens on the iximiuz
  playground CNI.
- Out of scope on purpose: image *building* (CKAD) — no container tooling in
  the lesson shell. (helm hands-on was here too — resolved 2026-07-13: lessons
  that need helm/trivy/cosign install a pinned binary in init_scenario.)
- Third-party installs (ArgoCD, Kyverno, Gatekeeper, KEDA, VPA, Flux, Capsule,
  Crossplane, prometheus-operator) use pinned official YAML manifests, never
  `helm install`.
- Dedup record (2026-07-13, gap-analysis pass — do NOT re-add): standalone
  CircleCI + Heroku reading lessons (already covered via upgrade-runbook /
  incident-datadog-cilium + case studies); pattern-pdb-eviction-block (dup of
  11.pdb-blocks-drain); cni-troubleshooting lab (dup of networkpolicy-blackhole
  → folded into cni-basics reading); api-server-apf reading (watch-informers
  covers APF); platform-slo-dashboards (merged into M8 slo-errorbudget).

## Cert coverage targets after backlog ships
| Cert | Before | Target |
|------|--------|--------|
| KCNA | ~65% | ~80% |
| CKAD | ~88% | ~98% (helm + ephemeral containers fill the last gaps) |
| CKA  | ~75% | ~90% (kubeadm/HA/cert-rotation are readings + iximiuz) |
| CKS  | ~55% | ~80% (encryption/audit/Falco are runbook readings by policy) |
| CNPE | ~20% | ~70% (Module 10 fills the platform gap) |

## Conventions to keep
- Verify scripts: fail with an actionable `not yet:` message; print `PASS — …`
  on success. Never wait on rollout for crashlooping-by-design pods.
- Keep solutions honest (root cause / fix / prevention); tie back to earlier
  lessons and the Incident Library where it reinforces the mental model.
