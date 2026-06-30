# Security model

Kubelings runs broken-on-purpose Kubernetes scenarios locally. The threat we
defend against: **lesson content must not be able to compromise the host
machine.** Lesson `init`/`verify` task scripts are authored content and are
treated as **untrusted code**.

## Trust boundary

| Component | Trust | Where it runs |
|-----------|-------|---------------|
| Lesson `tasks.*.run` scripts | **untrusted** | inside the kind **node container**, never on the host |
| `kind` create/delete (cluster lifecycle) | trusted (our scripts) | host |
| The TUI / Go code | trusted | host |
| Your `t` shell | **you** (the human) | host — it's your own shell, not a sandbox |

## Controls

1. **Lesson scripts execute in the kind node, not the host.** The runner runs
   every task `run:` block via `docker exec` inside the control-plane node
   container with the node's in-cluster kubeconfig. The host filesystem and
   processes are not visible to lesson code (verified: `/Volumes`, `/Users`
   inaccessible). See `_in_node` in `scripts/run-challenge-local.sh`.

2. **Path confinement.** The runner only executes an `index.md` resolved to a
   path **under `courses/kubelings/`** (real path checked; `..`/symlink escapes
   rejected). It cannot be pointed at an arbitrary file on disk. See `_confine`.

3. **Cluster confinement.** All cluster operations target the kind node's own
   admin kubeconfig — never the host's `KUBECONFIG`/current-context. A stray
   context pointing at a real/production cluster cannot be touched, and
   `reset`'s namespace delete can only affect the kind cluster.

4. **Pod Security `baseline` on the lesson namespace.** After init/reset the
   runner labels the `kubelings` namespace
   `pod-security.kubernetes.io/enforce=baseline`, so an untrusted lesson manifest
   cannot create `privileged`, `hostPath`, `hostNetwork`, or `hostPID` pods — the
   pod→node→host escape vectors (verified: such pods are `Forbidden`).

5. **Pinned supply chain.** Remote manifests are pinned to a version (e.g.
   metrics-server `v0.7.2`), not `latest`.

6. **The interactive shell is isolated from your global kube context.** `t`
   (and `kubelings shell`) use a dedicated temp `KUBECONFIG` exported from kind,
   so they never mutate your `~/.kube/config` current-context.

## Residual risks (by design / out of scope)

- **kind itself** runs each node as a privileged container on your Docker host.
  A container-runtime/kernel breakout from the node is theoretically possible —
  this is kind's inherent model, not specific to kubelings. Run untrusted course
  forks in a VM if that matters to you.
- **The `t` shell runs on the host as you.** It is your own shell for solving the
  scenario; it is intentionally not sandboxed.
- **Lesson content is still code.** Node-confinement + PSA limit the blast radius
  to the kind cluster. Only run courses you trust; review `tasks.*.run` before
  running third-party lessons.

## Reporting

Open a GitHub issue (or security advisory) on `madhank93/kubelings`.
