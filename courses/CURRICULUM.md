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

## Module 3 — Config & Storage
- [x] 1.configmap-wiring · [x] 2.secret-not-mounted · [x] 3.pvc-pending
- [x] 4.access-modes · [x] 5.pattern-pvc-terminating [DRILL]
- [x] 6.kustomize-overlays
- [ ] backlog: helm-releases (needs helm binary guaranteed in the lesson shell)

## Module 4 — Networking
- [x] 1.incident-dns-ndots [REAL Zalando] · [x] 2.networkpolicy-blackhole
- [x] 3.broken-targetport · [x] 4.nodeport-vs-clusterip
- [x] 5.incident-conntrack [REAL loveholidays/Preply, reading]
- [x] 6.incident-graceful-shutdown [REAL Ravelin] · [x] 7.ingress-wiring
- [x] 8.gateway-api (init fetches pinned CRDs — needs internet from cplane-01)
- [x] 9.kube-proxy-dataplane [reading: iptables/IPVS/nftables mechanics]

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

## Module 7 — Internals
- [x] 1.reconcile-loop · [x] 2.scheduler-nodename · [x] 3.etcd-truth
- [x] 4.control-plane-tour [reading: api-flow, kubelet/CRI, leader election]
- [x] 5.crd-operators · [x] 6.etcd-backup-restore [reading: snapshot/restore runbook]
- [x] 7.admission-mutations · [x] 8.watch-informers [reading, incl. APF]
- [x] 9.build-an-operator [reading: controller walkthrough capstone]
- [ ] backlog: CNI/CSI hands-on

## Module 8 — Observability & SRE
- [x] 1.events-forensics · [x] 2.incident-node-oom [REAL Blue Matador]
- [x] 3.quota-exhausted · [x] 4.node-notready · [x] 5.pattern-disk-pressure [DRILL]
- [x] 6.incident-datadog-cilium [REAL Datadog, reading]
- [x] 7.upgrade-runbook [reading: version skew, kubeadm, drain/uncordon]

## Module 9 — War Stories (capstone)
- [x] 1.incident-monzo-cascade [REAL Monzo, reading]
- [x] 2.incident-openai-cascade [REAL OpenAI, reading]
- [x] 3.incident-reddit-piday [REAL Reddit, reading]
- [x] 4.incident-black-friday [REAL Algolia, reading]
- [x] 5.final-boss [multi-fault, no hints]

## Incident library (docs)
- [x] Index seeded with 39 verified [REAL] rows + 6 [PATTERN] stubs; runnable/reading
      lessons cross-linked
- [x] Case studies: Zalando DNS, [PATTERN] PVC Terminating
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
  the lesson shell; helm hands-on — helm binary not guaranteed (kustomize
  lesson covers the comparison).

## Conventions to keep
- Verify scripts: fail with an actionable `not yet:` message; print `PASS — …`
  on success. Never wait on rollout for crashlooping-by-design pods.
- Keep solutions honest (root cause / fix / prevention); tie back to earlier
  lessons and the Incident Library where it reinforces the mental model.
