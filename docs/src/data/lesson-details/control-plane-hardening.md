> **Reading.** These controls are configured on hosts, in CI, or at cluster
> build time — outside this course's kubectl-only sandbox. But they complete
> the security model, and kube-bench (previous lesson) will keep flagging them
> until you know what they are. Six controls, each: *what it stops, where it
> lives, the one command or config that matters.*

## The map

Module 6 so far secured the layers kubectl can reach:

```
API access      → RBAC (6.1) + ServiceAccounts (6.5)
admission       → Pod Security Standards (6.4), webhooks (6.3)
pod/kernel      → securityContext hardening (6.6)
audit           → CIS benchmark (6.7)
```

This reading covers what's left — the layers *under* and *before* the pod.

## 1 · Secrets encryption at rest

**What it stops:** anyone with a copy of etcd (a stolen backup, a misplaced
snapshot, the disk itself) reading every Secret in plaintext. You proved
Secrets are just base64 in lesson 3.2; in etcd they're stored the same way
*unless you say otherwise*.

**Where:** an `EncryptionConfiguration` file on control-plane nodes, wired via
`kube-apiserver --encryption-provider-config`. Providers in order:
`aescbc`/`secretbox` (local keys — now etcd is safe but the key file on the
node is the new prize), or **KMS v2** (envelope encryption; the key never
touches the node — the production answer on any cloud).

**The gotcha everyone hits:** turning it on encrypts *new* writes only.
Existing Secrets stay plaintext until rewritten:
`kubectl get secrets -A -o json | kubectl replace -f -` — the documented
migration step people forget, leaving half-encrypted stores.

*Deep dive → lesson 6.16 `encryption-at-rest`: the full runbook, etcdctl
proof, and lockout-free key rotation.*

## 2 · Audit logging

**What it stops:** flying blind after a breach. The cryptominer incident
(6.2) response hinged on "what did that token *do*?" — without audit logs,
that question has no answer.

**Where:** `kube-apiserver --audit-policy-file` + `--audit-log-path`. The
policy language rules on *who/what/level*: `Metadata` for reads, `Request`
for writes, `RequestResponse` for secrets-adjacent verbs — and mute the
high-volume noise (kubelet status updates, leader-election leases) or the
log drowns itself. Ship them off-node (remember Datadog, M8: the observer
must not share fate with the observed).

**On-call reflex it enables:** `who deleted that namespace?` becomes a grep.

*Deep dive → lesson 6.17 `audit-policy`: a production-shaped policy file,
the apiserver wiring, and the greps that answer incident questions.*

## 3 · API server flags (the kube-bench section-1 canon)

Each of these is one flag on the API server static pod, and each is a
finding you saw (or will see) in kube-bench:

- `--anonymous-auth=false` — no unauthenticated API calls (kubelet too:
  anonymous kubelet = RCE on the node).
- `--authorization-mode=Node,RBAC` — never `AlwaysAllow`; `Node` scopes each
  kubelet to its own node's objects, so one stolen node credential can't
  read every Secret in the cluster.
- `--enable-admission-plugins=NodeRestriction,...` — enforces that node scoping.
- No `--insecure-port`, TLS everywhere, `--profiling=false`.

These live in `/etc/kubernetes/manifests/kube-apiserver.yaml` — a **static
pod** (M7): the kubelet restarts the API server the moment the file changes,
which is both how you apply these and how you cause a control-plane outage
with a typo. Change one flag at a time; on managed clusters, most of this
section is the provider's job (that's much of what you're paying for).

## 4 · Image supply chain

**What it stops:** running an attacker's code because it was *built into* the
image — the compromise that happens before your cluster ever sees a pod.

Three controls, cheap to expensive:

- **Pin by digest**: `nginx@sha256:...` instead of `:1.27-alpine`. Tags are
  mutable pointers — a compromised registry can re-point one silently; a
  digest can't lie. (Also kills the `:latest` drift you debugged in 1.6.)
- **Scan in CI**: trivy/grype against every image at build + a registry
  re-scan on a schedule (new CVEs land in *old* images). Gate merges on
  criticals with a documented-exception path, or the gate dies of politics.
- **Sign and verify**: cosign signatures checked by an admission webhook
  (6.3's mechanism, pointed at provenance) — only images your CI signed may
  run. Policy engines (Kyverno, Gatekeeper) express this as data.

The order matters: digests today, scanning this quarter, signing when the
first two are boring.

## 5 · Runtime security (Falco and friends)

**What it stops:** the attack you didn't prevent. Everything above is
*pre-execution*; runtime security watches syscalls *while pods run* — "shell
spawned in a production container", "pod read /etc/shadow", "outbound
connection to a mining pool" (exactly the 6.2 detection story).

**Where:** Falco (or Tetragon/Tracee) as a DaemonSet with eBPF, node-level
because syscalls are node-level. Rules are the product: start from the stock
ruleset, tune the noisy 10%, and *route alerts to a human within minutes* —
a runtime alert nobody reads is a postmortem footnote.

*Deep dive → lesson 6.18 `falco-runtime-detection`: architecture, the rules
language, and the shell-in-container rule end to end.*

## 6 · Sandboxed runtimes

**What it stops:** kernel exploits. Hardened or not, every container on a
node shares its kernel; a container-escape zero-day beats every pod field.
gVisor (userspace kernel) and Kata (microVM per pod) put a boundary between
container and host kernel.

**Where:** a `RuntimeClass` object + `runtimeClassName` in the pod spec —
but the runtime binary must exist on the node (containerd config), so this
is a node-image decision. Reserve it for genuinely untrusted or multi-tenant
workloads; it costs performance and compatibility.

## Who owns what (the honest table)

| Control | Self-managed | EKS/GKE/AKS |
|---|---|---|
| encryption at rest | you (flags/KMS) | checkbox / default |
| audit logs | you (policy + shipping) | mostly checkbox, you route |
| API server flags | you | provider |
| supply chain | **you** — always | **you** — always |
| runtime security | you (DaemonSet) | you (DaemonSet) |
| sandboxed runtime | you (node image) | provider-specific (gVisor on GKE) |

The two rows that never move: supply chain and runtime security are yours on
every platform. If you're on managed Kubernetes and wondering where to spend
security effort — it's rows 4 and 5, plus everything this module already
taught inside the cluster.

*No check — study, then advance.*
