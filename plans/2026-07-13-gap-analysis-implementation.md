# Gap-Analysis Implementation Plan — 35 lessons, Module 10, incident-library growth

> Source spec: `/Users/madhan/Downloads/kubelings-gap-analysis-for-claude-code.md` (2026-07-13)
> Each phase is self-contained: run it in a fresh context, read the cited files first.
> Course grows 72 → 107 lessons.

---

## Phase 0 — Findings & Dedup Decisions (already resolved — do not re-litigate)

### Repo facts (verified 2026-07-13)

| Fact | Evidence |
|---|---|
| Lessons run on iximiuz playground `k8s-omni`, machine `cplane-01`, init as root. There is NO `docker exec` into kind nodes — spec sections assuming local kind are wrong for the published course. | every existing `index.md` frontmatter |
| Local dev/test harness: `just run <slug> init` / `just run <slug> verify` against a local kind cluster via `scripts/run-challenge-local.sh`; scaffold via `scripts/scaffold-lesson.sh`; validators in `scripts/validators/` | `justfile`, `scripts/` |
| **Security confinement policy**: lesson scripts are kubectl-only by design. Host-level topics (apiserver manifest edits, kubeadm exec, AppArmor, Falco install) ship as **readings with full runbooks** | `courses/CURRICULUM.md` "Cert coverage notes" |
| Playground is multi-node (existing lessons iterate `kubectl get nodes`, cordon/drain) | `module-2/11.pdb-blocks-drain/index.md` |
| Internet available from cplane-01 (gateway-api init fetches pinned CRDs) | CURRICULUM.md M4 note |
| helm/trivy/cosign binaries NOT guaranteed in lesson shell → init_scenario must install pinned binaries itself (runs as root, has internet) | CURRICULUM.md "Out of scope" note |
| NetworkPolicy objects are verified but not enforced on local kind (kindnet); call out in units | CURRICULUM.md |
| TUI derives lesson badge from dir prefix (`incident-`/`pattern-`) + presence of `tasks:`; readings show links, not inline content | CURRICULUM.md header |
| Slug = dir basename minus `N.` prefix; progress keyed by slug → renumbering dirs is safe, renaming slugs is not | CURRICULUM.md header |

### Dedup decisions (6 spec items dropped/merged; 41 → 35 new lessons)

| Spec item | Decision | Reason |
|---|---|---|
| `incident-circleci-version-skew` (M8 reading) | **DROP** | Fully shipped 2026-07-08: case study `docs/src/content/docs/incidents/circleci-version-skew.md`, library row (line ~64), embedded in `module-8/7.upgrade-runbook/unit-1.md:30` |
| `incident-heroku-host-update` (M8 reading) | **DROP lesson, ADD case study page** | Library row exists (line ~65), cross-linked from `incident-datadog-cilium` which already teaches this failure mode. Missing only `docs/src/content/docs/incidents/heroku-host-update.md` |
| `pattern-pdb-eviction-block` (M2 drill) | **DROP** | Duplicate of existing `module-2/11.pdb-blocks-drain` — identical scenario (minAvailable == replicas, drain hangs) |
| `cni-troubleshooting` (M4 lab) | **MERGE into `cni-basics` reading** | Spec scenario (empty-podSelector deny-all) duplicates existing `module-4/2.networkpolicy-blackhole`. Real CNI debugging is host-level (crictl, conflist) → confinement policy says reading |
| `platform-slo-dashboards` (M10 lab) | **MERGE into M8 `slo-errorbudget`** | Both are "fix broken PrometheusRule burn-rate expr". One lab, two bugs (wrong metric name + 4xx-in-5xx expr) |
| `api-server-apf` (M7 reading) | **DROP** | `module-7/8.watch-informers` reading already includes APF (per CURRICULUM.md). Phase 6 checks it mentions FlowSchema/PriorityLevelConfiguration + CircleCI/OpenAI links; enrich in place if thin |
| `encryption-at-rest`, `audit-policy` (M6 labs) | **DEMOTE to runbook readings** | Require apiserver static-pod edits = host-level = confinement policy. Precedent: `6.etcd-backup-restore [reading: runbook]`. `control-plane-hardening` covers these only survey-level (~15 lines each) so dedicated runbooks still add value; cross-link both ways |
| `falco-runtime-detection` (M6) | Reading with runbook (as spec suggests) | kernel module/eBPF = host-level |
| `cert-rotation` (M7) | Reading with runbook | `kubeadm certs renew` = host-level |
| `node-maintenance` (M7/M8 cross-listed) | **Place in M8** (SRE operational procedure) | cordon/drain/uncordon is kubectl-only → full lab OK. Distinct from pdb-blocks-drain (DaemonSet + emptyDir flags, full maintenance cycle) |
| `apparmor-seccomp` | Keep, as spec's two-unit split | unit-1 seccomp `RuntimeDefault` is kubectl-only (verified: `container-hardening` does NOT cover seccomp); unit-2 AppArmor is reading |
| Spotify library row | Fix module M8 → M9 | Row exists at library line ~30 pointing to M8; lesson lands in M9 |
| Weaveworks library row | Verify URL first — Weaveworks shut down 2024; use Wayback Machine link or drop row | CURRICULUM policy: "verify every URL before adding" |

### Final lesson numbering (assign now; phases are then order-independent)

- **M2**: 15.ephemeral-containers · 16.multi-container-patterns · 17.pattern-readiness-flap · 18.pattern-zombie-cronjobs · 19.pattern-rolling-update-deadlock · 20.vpa · 21.keda-autoscaling
- **M3**: 7.helm-releases · 8.pattern-ghost-endpoints · 9.pattern-secret-not-reloaded · 10.pattern-namespace-terminating
- **M4**: 10.cni-basics [reading] · 11.kubeconfig-contexts
- **M6**: 11.opa-gatekeeper · 12.kyverno-policies · 13.image-scanning-pipeline · 14.sbom-cosign · 15.apparmor-seccomp · 16.encryption-at-rest [reading] · 17.audit-policy [reading] · 18.falco-runtime-detection [reading]
- **M7**: 10.kubeadm-bootstrap [reading] · 11.ha-control-plane [reading] · 12.cert-rotation [reading]
- **M8**: 8.node-maintenance · 9.slo-errorbudget · 10.otel-collector-pipeline
- **M9**: 5.incident-target-cascade [reading] · 6.incident-spotify-delete [reading] · **renumber** `5.final-boss` → `7.final-boss` (dir rename only; slug `final-boss` unchanged, progress safe)
- **M10 (new)**: 1.gitops-argocd · 2.gitops-argocd-appofapps · 3.gitops-flux2 · 4.multi-tenancy-capsule · 5.cluster-api-intro [reading] · 6.crossplane-compositions

### Exemplars to copy (read before authoring — do not invent structure)

| Authoring | Copy from |
|---|---|
| Lab lesson index.md + unit-1.md | `courses/kubelings/module-2/11.pdb-blocks-drain/` |
| Pattern drill | `courses/kubelings/module-3/5.pattern-pvc-terminating/` |
| Reading lesson | `courses/kubelings/module-4/9.kube-proxy-dataplane/` |
| Incident reading lesson (with `source:`) | `courses/kubelings/module-8/6.incident-datadog-cilium/` |
| Case study page | `docs/src/content/docs/incidents/circleci-version-skew.md` |
| Module index | `courses/kubelings/module-9/0.index.md` |
| Scaffold | `scripts/scaffold-lesson.sh <module> <slug>` (read it first; use if it fits) |

### Global anti-pattern guards (every phase)

- NO `sleep`-polling loops in verify; no waiting on crashlooping-by-design pods
- All learner resources in `namespace: kubelings`, created idempotently (`--dry-run=client -o yaml | kubectl apply -f -`)
- NO `docker exec` into nodes; NO editing `/etc/kubernetes/manifests` in tasks (readings may document those steps as runbooks)
- Failure paths: `echo "not yet: <actionable>"; exit 1`. Success: `echo "PASS — <what was proved>."`
- Binary installs in init: pinned versions, checksummed where practical, to `/usr/local/bin`
- Third-party installs (ArgoCD, Kyverno, …): pinned-version official YAML manifests, NOT `helm install` (helm only inside `helm-releases` where init installs it)
- Verify with structural assertions per house style: `kubectl auth can-i`, `jsonpath`, `kubectl get -o yaml | grep`
- Reading lessons: omit `tasks:` entirely; incident readings add `source: <url>`
- Every URL added anywhere must be fetched and confirmed live (or Wayback'd) before commit

### Per-lesson definition of done (all phases)

1. `index.md` + `unit-1.md` per spec schema (spec §1)
2. `just run <slug> init && just run <slug> verify` — verify fails before fix; after applying the documented Solution manually, verify passes (labs/drills only)
3. Module `0.index.md` bumped `updatedAt`
4. `courses/CURRICULUM.md` item flipped `[x]`
5. Incident readings: library row + case study page exist and cross-link

---

## Phase 1 — Bookkeeping, Module 10 scaffold, M9 readings & case studies

**Read first:** spec §4–5; `docs/src/content/docs/reference/incident-library.md`; `courses/CURRICULUM.md`; exemplar case study + incident-reading lesson (table above).

1. **CURRICULUM.md**: append backlog blocks for M2–M10 using the deduped numbering above (NOT spec §5 verbatim — it contains the 6 dropped items). Mark dropped spec items nowhere; they simply don't appear.
2. **Module 10 scaffold**: `courses/kubelings/module-10/0.index.md` (copy `module-9/0.index.md` shape; description from spec §"NEW MODULE 10").
3. **M9 renumber**: `git mv courses/kubelings/module-9/5.final-boss courses/kubelings/module-9/7.final-boss`. Grep repo for `5.final-boss` path references and fix.
4. **Case studies** (copy `circleci-version-skew.md` structure — frontmatter, timeline, root cause, lessons, citations):
   - `docs/src/content/docs/incidents/heroku-host-update.md` (source: heroku.com/blog/summary-of-june-10-outage + status.heroku.com/incidents/2822)
   - `docs/src/content/docs/incidents/target-cascade.md` (medium.com Daniel Woods post)
   - `docs/src/content/docs/incidents/spotify-delete.md` (KubeCon talk youtube.com/watch?v=ix0Tw8uinWs)
5. **M9 reading lessons** `5.incident-target-cascade`, `6.incident-spotify-delete`: reading lessons (no `tasks:`), `source:` URL, body links to the case-study page (TUI shows links, not inline content).
6. **Incident library edits**:
   - Fix Spotify row: module M8 → M9, add case-study link
   - Add Target case-study link to its existing row; add Heroku case-study link to its row
   - New rows: GKE/Google 19009 (verify status.cloud.google.com URL), Weaveworks GitOps divergence (verify URL, likely Wayback)
   - Pattern-stub table: add rows for HPA thrashing (M2), webhook timeout (M6/M7), mutable tag drift (M6), rolling-update deadlock (M2), namespace stuck Terminating (M3), watch-storm APF (M7), etcd compaction stall (M7); link "PDB blocks evictions" to existing `pdb-blocks-drain`

**Verify:** `just docs-build` passes; every added URL fetched live; `grep -rn "5.final-boss"` returns nothing; `go test ./...` still green (TUI parses course tree).

---

## Phase 2 — M2 kubectl-only labs & drills (5 lessons)

**Read first:** spec §3 M2 entries; exemplar lab + drill (Phase 0 table).

| Lesson | Init (broken state) | Verify (structural) |
|---|---|---|
| 15.ephemeral-containers | distroless pod, bogus config, exit-1 loop | `jsonpath {.spec.ephemeralContainers}` non-empty AND pod Running |
| 16.multi-container-patterns | 3 pods: sidecar wrong volume path, ambassador wrong port, adapter wrong output | per-pod structural checks (volume data, port, pipe) |
| 17.pattern-readiness-flap | probe `failureThreshold:1 periodSeconds:1`, slow container | probe thresholds sane AND pod Ready (single check — no 30 s wait loops; assert `lastTransitionTime` age via jsonpath if stability proof needed) |
| 18.pattern-zombie-cronjobs | CronJob history limits 50, pre-seeded completed Jobs (create Jobs directly with `status`-complete via short-running `spec.template` — do NOT wait 50 schedule ticks) | limits ≤ 3 AND `ttlSecondsAfterFinished` set |
| 19.pattern-rolling-update-deadlock | `maxUnavailable:0 maxSurge:0` + unschedulable new pod (oversized request) | `kubectl rollout status --timeout=60s` exits 0 |

Notes: ephemeral-containers `kubectl debug --target` requires `EphemeralContainers` — supported on modern clusters, confirm against local kind during test. Zombie-cronjobs: seed completed Jobs with `restartPolicy: Never`, command `true`, then wait for completion in init (init may wait; verify may not).

**Verify:** per-lesson definition of done; both broken→fail and fixed→pass paths exercised locally.

---

## Phase 3 — M3 drills + helm-releases (4 lessons)

**Read first:** spec §3 M3; drill exemplar.

- **7.helm-releases**: init installs pinned helm (`curl` official release tarball, pin version + sha256), `helm install` a small local chart (vendor a tiny chart under the lesson dir or `__static__/` — do NOT depend on bitnami repo availability), then `helm upgrade` with values that crashloop it. Verify: `helm status -o json` shows `deployed` AND pod Running. Update CURRICULUM.md "Out of scope" note: helm now in scope via init-installed binary.
- **8.pattern-ghost-endpoints**: init: Deployment `terminationGracePeriodSeconds: 0`, no preStop. Verify: preStop exec present AND `terminationGracePeriodSeconds >= 5` (structural — do NOT try to time 503 windows).
- **9.pattern-secret-not-reloaded**: init: Secret injected as env, then rotate value. Verify: pod consumes Secret via volume mount (jsonpath on `.spec.volumes` + mount) OR env-pod restarted after rotation (restartCount/newer startTime); prefer the mount assertion.
- **10.pattern-namespace-terminating**: init: create sacrificial namespace (NOT `kubelings`) with a custom-CRD instance carrying a finalizer, delete the CRD's controller-less finalizer owner, delete namespace → stuck Terminating. Verify: `kubectl get namespace <name>` returns NotFound.

**Verify:** definition of done ×4.

---

## Phase 4 — M4 additions (2 lessons)

**Read first:** spec §3 M4; reading exemplar; `module-4/2.networkpolicy-blackhole/` (to avoid re-teaching it).

- **10.cni-basics** [reading]: merged scope — CNI plugin dir + `conflist` anatomy, how kubelet finds the binary, Calico/Flannel install shape, `crictl` debugging, why node goes NotReady without CNI, AND a short troubleshooting decision tree (pod stuck ContainerCreating → describe → CNI vs NetworkPolicy triage; link `networkpolicy-blackhole` for the policy path). Satisfies M7 "backlog: CNI/CSI hands-on" reading half — note that in CURRICULUM.
- **11.kubeconfig-contexts**: init writes a 3-context kubeconfig to a known path (contexts: prod/staging/dev pointing at the real cluster server but distinct context+namespace names — no fake clusters, kubectl must keep working). Verify: `kubectl config current-context --kubeconfig <path>` = staging AND merged file (`KUBECONFIG=a:b kubectl config view --flatten`) written where task says.

**Verify:** definition of done ×2.

---

## Phase 5 — M6 security (8 lessons: 5 hands-on, 3 runbook readings)

**Read first:** spec §3 M6; `module-6/8.control-plane-hardening/unit-1.md` (cross-link, don't duplicate); reading + lab exemplars.

Hands-on (kubectl/client-side only):
- **11.opa-gatekeeper**: init installs pinned Gatekeeper manifest, ConstraintTemplate with Rego bug (`.image` vs `.spec.containers[_].image`), Constraint denying `:latest`. Verify: apply of `nginx:latest` pod fails (exit non-zero) AND pinned-tag pod succeeds.
- **12.kyverno-policies**: init installs pinned Kyverno manifest + Enforce ClusterPolicy matching ALL namespaces. Verify: pod creatable in `kube-system`; label-less pod in `kubelings` gets `app` label mutated in; `:latest` pod rejected. Keep webhook `failurePolicy` implications tied back to `incident-webhook-outage`.
- **13.image-scanning-pipeline**: init installs pinned trivy binary + Deployment with `nginx:1.14`. Verify: Deployment image matches `@sha256:` regex AND `trivy image --severity CRITICAL --exit-code 1 <image>` passes. Cross-link existing `10.image-digests`.
- **14.sbom-cosign** [reading + light lab]: init installs pinned cosign (+syft). Task: `cosign verify` a well-known keyless-signed public image; inspect SBOM. Verify: a marker file the learner writes with the verified digest matches `cosign verify` output digest (client-side assertions only).
- **15.apparmor-seccomp**: unit-1 (hands-on): add `seccompProfile: {type: RuntimeDefault}` to a pod; verify via jsonpath. unit-2 content (AppArmor runbook: `apparmor_parser`, annotation, `/proc/1/attr/current`) goes in the same unit-1.md as a clearly-marked reading section OR a `unit-2.md` if the course loader supports multi-unit (check `internal/` course parser first — existing lessons are all single-unit).

Runbook readings (no `tasks:`):
- **16.encryption-at-rest**: EncryptionConfiguration anatomy, aescbc vs kms, apiserver flag, `etcdctl` verification (`k8s:enc:aescbc` prefix), re-encrypt-all procedure. Cross-link control-plane-hardening §1 → this.
- **17.audit-policy**: policy YAML levels (None/Metadata/Request/RequestResponse), targeting secrets + pods/exec, apiserver flags, log inspection. Cross-link control-plane-hardening §2.
- **18.falco-runtime-detection**: architecture (kernel module vs eBPF), rules YAML, custom shell-in-container rule, install runbook + iximiuz/Linux note from spec. Cross-link control-plane-hardening §5.

Also: trim `control-plane-hardening` §§1/2/5 closers to point at the new deep readings (one-line "deep dive →" links; don't rewrite the survey).

**Verify:** definition of done ×8; Gatekeeper/Kyverno inits must be idempotent (re-run safe) and pinned; confirm the two admission installs don't fight each other if a learner runs both lessons in one session (scope Constraints/Policies to `kubelings` namespace, exclude system namespaces).

---

## Phase 6 — M7 readings (3 lessons) + watch-informers APF check

**Read first:** spec §3 M7; `module-7/8.watch-informers/` and `module-8/7.upgrade-runbook/` (dedup boundaries); reading exemplar.

- **10.kubeadm-bootstrap** [reading]: full `kubeadm init → join` runbook (pod-network-cidr, kubeconfig copy, token join, CNI install, node Ready). Explicit note: cannot run inside k8s-omni; practice on iximiuz multi-VM playground. Don't re-teach upgrade flow — link upgrade-runbook.
- **11.ha-control-plane** [reading]: stacked vs external etcd, `--control-plane-endpoint`, join --control-plane, leader election, VIP options; kind 3-CP config snippet as concept validation.
- **12.cert-rotation** [reading runbook]: `kubeadm certs check-expiration` / `renew`, which components restart, CA vs leaf certs, kubelet cert rotation flags.
- **APF check**: read `watch-informers` unit; if FlowSchema/PriorityLevelConfiguration mechanics + CircleCI/OpenAI incident links absent, enrich in place (small additive section, keep lesson focus).

**Verify:** docs/course builds; no `tasks:` blocks present; CURRICULUM flips.

---

## Phase 7 — M8 SRE (3 lessons)

**Read first:** spec §3 M8; lab exemplar; `module-2/11.pdb-blocks-drain` (differentiate).

- **8.node-maintenance**: init: pick a worker node, deploy DaemonSet + emptyDir pod onto it (nodeSelector). Task: cordon → drain `--ignore-daemonsets --delete-emptydir-data` → uncordon. Verify: node schedulable again AND a `maintenance-done` marker (e.g. learner labels the node) AND evicted pod rescheduled elsewhere — all via jsonpath; no timing loops. Unit explicitly contrasts with pdb-blocks-drain ("last time drain hung; this time drain right").
- **9.slo-errorbudget** (merged lesson): init installs pinned prometheus-operator bundle manifest (NOT kube-prometheus-stack Helm), a tiny synthetic-metrics app, and a PrometheusRule with BOTH bugs (wrong metric name; 4xx counted as 5xx). Verify: `kubectl get prometheusrule -o yaml` matches corrected exprs AND Prometheus HTTP API (`kubectl exec curl` against prom svc) returns non-zero burn-rate sample. Mind playground resources — single replica, no Grafana (dashboard JSON shown in unit as reading material instead).
- **10.otel-collector-pipeline**: init: pinned OTel Collector Deployment (not DaemonSet — lighter) + Jaeger all-in-one manifest; ConfigMap exporter endpoint pointing at wrong svc name. Verify: collector logs free of export errors (`kubectl logs | grep -c "export.*error" == 0` style) AND Jaeger query API returns the test trace (`kubectl exec curl` the jaeger-query svc).

**Verify:** definition of done ×3; watch total resource footprint — these three must not assume >2 GiB free on the playground; keep replicas=1 and requests small.

---

## Phase 8 — Module 10 Platform Engineering (6 lessons)

**Read first:** spec §3 M10; lab + reading exemplars. All installs = pinned official YAML manifests. GitOps sources = public upstream repos (document exact repo+path+revision in init comments).

- **1.gitops-argocd**: init installs pinned ArgoCD `install.yaml` (argoproj release), Application with wrong `spec.source.path` against `argoproj/argocd-example-apps` (guestbook). Verify: Application `.status.sync.status == Synced` AND `.status.health.status == Healthy` via jsonpath (kubectl, not argocd CLI — don't assume the binary).
- **2.gitops-argocd-appofapps**: parent Application + 3 children, one child wrong `targetRevision`. Verify: all 4 Applications Synced+Healthy. Teach sync-waves annotation in unit.
- **3.gitops-flux2**: init installs pinned Flux2 `install.yaml` (fluxcd release — NOT `flux bootstrap`, which wants repo write access), GitRepository pointing at `stefanprodan/podinfo` wrong branch + Kustomization. Verify: Kustomization `.status.conditions[?(@.type=="Ready")].status == True`.
- **4.multi-tenancy-capsule**: init installs pinned Capsule manifest; Tenant `team-alpha` with nonexistent nodeSelector label. Verify: pod schedules in team-alpha namespace AND cross-tenant access denied (`kubectl auth can-i --as` the tenant owner against team-beta ns == no).
- **5.cluster-api-intro** [reading]: CAPI architecture, provider model, MachineDeployment YAML examples, CAPI vs kubeadm; no live cluster needed.
- **6.crossplane-compositions**: init installs pinned Crossplane manifest + XRD + Composition referencing a provider that isn't installed (use `provider-nop` or `provider-kubernetes` as the fix — verify which is lightest at authoring time). Verify: XR `.status.conditions Ready == True`.

Update `docs/src/content/docs/guides/lessons.md` catalog + curriculum guide with Module 10.

**Verify:** definition of done ×6; each init idempotent + re-runnable; ArgoCD/Flux/Capsule/Crossplane pinned versions recorded in init comments; total install size sanity-checked on local kind first.

---

## Phase 9 — M2 install-heavy autoscaling (2 lessons)

**Read first:** spec §3 M2 vpa/keda entries; existing `module-2/6.hpa/` (cross-link, differentiate).

- **20.vpa**: init installs VPA from `kubernetes/autoscaler` pinned manifests (recommender only — skip updater/admission-controller to stay light), Deployment + VPA object with `minAllowed > maxAllowed` nonsense bounds, `updateMode: "Off"` vs correct mode taught in unit. Verify: `kubectl get vpa -o jsonpath='{.status.recommendation}'` non-empty. NOTE: recommender needs metrics history — confirm on local kind how long until first recommendation; if >2 min, seed with a busy pod in init and set `--recommender-interval` low via the pinned manifest patch.
- **21.keda-autoscaling**: init installs pinned KEDA YAML (keda release, not Helm); metric source: KEDA `cron` or `cpu` trigger misconfigured — PREFER a trigger that needs no Prometheus install (spec's Prometheus trigger drags a whole stack into an M2 lesson; `cpu`/`cron` trigger teaches the same ScaledObject mechanics). Verify: `kubectl get scaledobject -o jsonpath` READY True AND HPA created by KEDA exists.

**Verify:** definition of done ×2; both inits < 1 min on local kind.

---

## Phase 10 — Final verification & ship

1. `go build ./... && go test ./...`
2. `just doctor` — course tree parses, all 107 lessons listed, badges correct (replay/drill/read/lab derived from prefix+tasks)
3. Slug uniqueness: `find courses/kubelings -maxdepth 2 -type d | sed 's/.*[0-9]*\.//' | sort | uniq -d` → empty
4. `just docs-build` — docs site builds; incident-library links resolve
5. Validators: run whatever `scripts/validators/` provides over all new lessons
6. Full local pass: for each new lab/drill slug → `just run <slug> init && just run <slug> verify` (expect fail) → apply Solution → verify (expect PASS)
7. CURRICULUM.md: all new items `[x]`; cert-coverage table from spec §5 appended (with note that CKS/CKA percentages reflect the runbook-reading demotions)
8. Grep anti-patterns: `grep -rn "sleep " courses/kubelings/module-*/1[0-9].*/index.md` and new dirs → no polling sleeps in verify blocks; `grep -rln "docker exec" courses/` → readings only
9. Update `docs/src/content/docs/guides/lessons.md` totals + module list; bump all touched module `0.index.md` `updatedAt`
10. Commit per phase or per module (Conventional Commits, as repo history does); push after full pass; deploy to iximiuz per existing flow (labctl content push) — course dir must contain no non-course markdown (CURRICULUM.md validation incident, 2026-07-08)
