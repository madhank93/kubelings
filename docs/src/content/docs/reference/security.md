---
title: Security
description: The trust model — lesson code cannot compromise your host.
---

The threat Kubelings defends against: **lesson content must not be able to
compromise the host machine.** Lesson `init`/`verify` task scripts are authored
content and are treated as **untrusted code**.

## Trust boundary

| Component | Trust | Where it runs |
|-----------|-------|---------------|
| Lesson `tasks.*.run` scripts | **untrusted** | inside the kind **node container** |
| `kind` create/delete | trusted (our scripts) | host |
| The TUI / Go code | trusted | host |
| Your `t` shell | **you** (the human) | host — your own shell |

## Controls

1. **Lesson scripts execute in the kind node, not the host.** Every task `run:`
   block runs via `docker exec` inside the control-plane node container with the
   node's in-cluster kubeconfig. The host filesystem/processes are invisible to
   lesson code.
2. **Path confinement.** Only an `index.md` resolved under `courses/kubelings/`
   is executed; `..`/symlink escapes are rejected and the run aborts.
3. **Cluster confinement.** All cluster ops target the node's admin kubeconfig,
   never your host `KUBECONFIG`/current-context — a stray context (e.g. prod)
   can't be touched, and `reset`'s namespace delete stays inside the kind cluster.
4. **Pod Security `baseline`** is enforced on the `kubelings` namespace, blocking
   `privileged` / `hostPath` / `hostNetwork` / `hostPID` pods — the
   pod→node→host escape vectors.
5. **Pinned supply chain** — remote manifests use pinned versions, not `latest`.
6. **Isolated shell KUBECONFIG** — the interactive shell uses a temp kubeconfig
   exported from kind, so it never mutates your `~/.kube/config`.

## Residual risks (by design)

- **kind** runs each node as a privileged container on Docker; a runtime/kernel
  breakout from the node is theoretically possible — kind's inherent model. Run
  untrusted forks in a VM if that matters to you.
- **The `t` shell runs on the host as you** — intentionally not sandboxed.
- **Lesson content is still code** — confinement limits the blast radius to the
  kind cluster. Only run courses you trust.

The canonical, always-current version lives in
[`SECURITY.md`](https://github.com/madhank93/kubelings/blob/main/SECURITY.md).
