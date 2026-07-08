---
title: Lessons
description: The full Kubelings lesson catalog (72 lessons, 9 modules) and how to add a new one.
---

Lessons live under `courses/kubelings/module-N/<n>.<name>/`. Each has:

- `index.md` — frontmatter with the `playground` and `init`/`verify` **tasks**.
- `unit-1.md` — the prose: situation, task, hint, the interactive check, and the
  solution (in a collapsible block).

Four lesson types (the TUI shows a badge for each):

| Type | Meaning |
|---|---|
| **lab** | hands-on concept lesson, verify-gated |
| **replay** | a real, cited production incident, reproduced |
| **drill** | a synthetic failure pattern seen across many companies |
| **read** | guided reading (runbooks, incident files) — no tasks |

## Module 1 — Foundations

| Lesson | Scenario | Type |
|--------|----------|------|
| `welcome` | tour the tooling and the loop | read |
| `crashloop-triage` | diagnose and fix a CrashLoopBackOff | lab |
| `expose-web` | expose a deployment with a Service | lab |
| `selector-mismatch` | service selects nothing — fix the labels | lab |
| `namespace-basics` | find workloads across namespaces | lab |
| `imagepull-backoff` | fix an ImagePullBackOff (bad tag) | lab |
| `kubectl-detective` | find the hidden fault with only kubectl | lab |

## Module 2 — Workloads

| Lesson | Scenario | Type |
|--------|----------|------|
| `rolling-update` | fix unsafe `maxSurge`/`maxUnavailable` | lab |
| `daemonset` | build a node-level log collector | lab |
| `statefulset` | StatefulSet + headless Service (stable identity) | lab |
| `jobs` | make a never-finishing Job complete | lab |
| `cronjobs` | stop CronJob pileup via `concurrencyPolicy` | lab |
| `hpa` | autoscale a Deployment with an HPA (1→5) | lab |
| `oomkill` | right-size memory to stop an OOMKill loop | lab |
| `liveness-vs-readiness` | the probe that restarts vs the probe that gates traffic | lab |
| `startup-probe` | stop killing a slow-starting app | lab |
| `init-containers` | debug an init container failing silently | lab |
| `pdb-blocks-drain` | the drain that never finishes (PDB math) | lab |
| `qos-classes` | Guaranteed / Burstable / BestEffort and who dies first | lab |
| `blue-green-canary` | flip traffic with one selector change; keep the way back | lab |
| `incident-cpu-throttling` | Omio/Buffer's latency tax — read cpu.stat, fix the limit | replay |

## Module 3 — Config & Storage

| Lesson | Scenario | Type |
|--------|----------|------|
| `configmap-wiring` | ConfigMap key mismatch starves the app of config | lab |
| `secret-not-mounted` | secret volume that never mounts — diagnose why | lab |
| `pvc-pending` | PVC stuck Pending (StorageClass mismatch) | lab |
| `access-modes` | RWO vs RWX and the pod that can't schedule | lab |
| `pattern-pvc-terminating` | PVC stuck Terminating — finalizers, done safely | drill |
| `kustomize-overlays` | kill config drift with base + prod overlay (`kubectl apply -k`) | lab |

## Module 4 — Networking

| Lesson | Scenario | Type |
|--------|----------|------|
| `incident-dns-ndots` | Zalando's total DNS outage — ndots:5 amplification | replay |
| `networkpolicy-blackhole` | default-deny blackholed the app — write the allows | lab |
| `broken-targetport` | service port vs targetPort vs containerPort | lab |
| `nodeport-vs-clusterip` | pick and wire the right Service type | lab |
| `incident-conntrack` | conntrack table exhaustion (loveholidays, Preply) | read |
| `incident-graceful-shutdown` | Ravelin's 502s — termination vs endpoints race | replay |
| `ingress-wiring` | ingress backend name+port chain, two breaks | lab |
| `gateway-api` | GatewayClass → Gateway → HTTPRoute, the app team's half | lab |
| `kube-proxy-dataplane` | there is no proxy — iptables, IPVS, and the DNAT dice roll | read |

## Module 5 — Scheduling & Placement

| Lesson | Scenario | Type |
|--------|----------|------|
| `incident-same-node` | Moonlight's outage — all replicas on the node that died | replay |
| `taints-tolerations` | keep workloads off (and on) the right nodes | lab |
| `topology-spread` | balance replicas across nodes proportionally | lab |
| `incident-priority-preemption` | Grafana's outage — preemption evicts production | replay |
| `pattern-noisy-neighbor` | one tenant starves the node — diagnose contention | drill |

## Module 6 — Security

| Lesson | Scenario | Type |
|--------|----------|------|
| `rbac-least-privilege` | shrink a god-mode grant to a least-privilege Role | lab |
| `incident-cryptominer` | JW Player's miner — find the pod nobody deployed | replay |
| `incident-webhook-outage` | Jetstack's GKE outage — webhook blocks every write | replay |
| `pod-security-baseline` | enforce Pod Security Standards on a namespace | lab |
| `serviceaccount-tokens` | the token in every pod — own identity, automount off | lab |
| `container-hardening` | non-root, read-only FS, zero caps, seccomp — still working | lab |
| `cis-kube-bench` | run the CIS benchmark as a Job and read the verdicts | lab |
| `control-plane-hardening` | encryption at rest, audit, API flags, supply chain, runtime | read |
| `egress-lockdown` | deny-all egress + DNS and one business flow back | lab |
| `image-digests` | tags lie — pin the deployment to a verified digest | lab |

## Module 7 — Internals

| Lesson | Scenario | Type |
|--------|----------|------|
| `reconcile-loop` | watch controllers converge spec → status | lab |
| `scheduler-nodename` | bypass and understand the scheduler pipeline | lab |
| `etcd-truth` | etcd is the cluster — prove it | lab |
| `control-plane-tour` | request flow, kubelet ↔ CRI, leader election | read |
| `crd-operators` | teach the API server a new noun (CRD + CR) | lab |
| `etcd-backup-restore` | the etcdctl snapshot/restore runbook | read |
| `admission-mutations` | watch admission rewrite your YAML (LimitRange, both halves) | lab |
| `watch-informers` | list+watch, resourceVersion, informers, APF | read |
| `build-an-operator` | the 60 lines that make an operator — annotated | read |

## Module 8 — Observability & SRE

| Lesson | Scenario | Type |
|--------|----------|------|
| `events-forensics` | reconstruct what happened from events alone | lab |
| `incident-node-oom` | Blue Matador's node OOM — kernel killer vs kubelet | replay |
| `quota-exhausted` | deploy stuck at 2/5 — the error is on the ReplicaSet | lab |
| `node-notready` | the morning after NotReady — taints and leftovers | lab |
| `pattern-disk-pressure` | Evicted: the disk you forgot to budget | drill |
| `incident-datadog-cilium` | the OS update that wiped CNI routes fleet-wide | read |
| `upgrade-runbook` | version skew, kubeadm sequence, drain → uncordon | read |

## Module 9 — War Stories (capstone)

| Lesson | Scenario | Type |
|--------|----------|------|
| `incident-monzo-cascade` | etcd + mesh + empty endpoints stop a bank | read |
| `incident-openai-cascade` | telemetry rollout locks operators out of the fix | read |
| `incident-reddit-piday` | one renamed label collapses the pod network | read |
| `incident-black-friday` | Jobs overload kills the dashboard at peak | read |
| `final-boss` | three faults, no hints | lab |

## Running a lesson

```sh
scripts/run-challenge-local.sh <lesson> init     # build it
scripts/run-challenge-local.sh <lesson> verify   # check your fix
scripts/run-challenge-local.sh <lesson> reset    # wipe + re-init
scripts/run-challenge-local.sh <lesson> solution # print the lesson (incl. solution)
```

`<lesson>` accepts a lesson name (e.g. `rolling-update`), its slug, or a dir path
(confined to the course tree).

## Adding a lesson

```sh
tools/scaffold-lesson.sh module-2 14 my-lesson "Fix the broken thing" k8s-omni
# edit courses/kubelings/module-2/14.my-lesson/{index.md,unit-1.md}

# test locally
scripts/run-challenge-local.sh my-lesson init && scripts/run-challenge-local.sh my-lesson verify

# publish the whole course
labctl content push course kubelings-dbd840c8 --dir courses/kubelings --force
```

### Task authoring rules

- `init` tasks build the broken/baseline state; `verify` exits `0` when solved,
  non-zero otherwise.
- Keep checks plain `kubectl` + bash.
- Lesson scripts run **inside the kind node**, not on your host — see
  [Security](/reference/security/). Avoid `hostPath`/privileged pods; the lesson
  namespace enforces Pod Security `baseline`.
- Incident replays carry a verified `source:` URL; see the
  [Incident Library](/reference/incident-library/) for the index.
