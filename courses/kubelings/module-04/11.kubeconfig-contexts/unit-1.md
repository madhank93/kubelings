---
kind: unit
title: "kubeconfig: contexts, the merge, and the prod you almost touched"
name: kubeconfig-contexts-unit
---


## The situation

Init dropped two kubeconfig files in `/tmp/kubelings-kubeconfig/`:

```sh
DIR=/tmp/kubelings-kubeconfig
kubectl config get-contexts --kubeconfig $DIR/main
# CURRENT   NAME      CLUSTER   NAMESPACE
# *         prod      …         default
#           staging   …         kubelings
#           dev       …         dev
kubectl config get-contexts --kubeconfig $DIR/extra
#           observability   …   monitoring
```

A **context** is just a named triple: *cluster* (server + CA) + *user*
(credentials) + default *namespace*. The file's `current-context` decides
where every bare `kubectl` command lands — and right now it's `prod`. Every
`kubectl delete` you type without `--context` goes to production. This exact
default has starred in several of this course's incident files (Spotify's
cluster deletion began with a terminal pointed at the wrong cluster).

## Your task

All against the files in `$DIR` (leave your real `~/.kube/config` alone):

1. **Switch off prod** — make `staging` the current context of `$DIR/main`:

   ```sh
   kubectl config use-context staging --kubeconfig $DIR/main
   ```

2. **Merge** the teammate's `extra` file into a single self-contained file at
   `$DIR/merged`, keeping all four contexts. The merge operator is the
   `KUBECONFIG` path list plus `--flatten`:

   ```sh
   KUBECONFIG=$DIR/main:$DIR/extra kubectl config view --flatten > $DIR/merged
   ```

3. Prove the merged file works:

   ```sh
   kubectl --kubeconfig $DIR/merged config get-contexts
   kubectl --kubeconfig $DIR/merged --context=staging get pods
   ```

<details>
<summary>Hint</summary>

`--flatten` matters: plain `kubectl config view` redacts certificate data
(`DATA+OMITTED`), and a merged file full of redactions can list contexts but
can't authenticate. `--flatten` inlines every cert and key so the file stands
alone. Precedence in `KUBECONFIG=a:b`: first file wins conflicts —
including `current-context`.

</details>

::simple-task
---
:tasks: tasks
:name: verify_done
---
#active
Solve the task above — this check turns green once verification passes.

#completed
✅ Solved — nicely done!
::

<details>
<summary>Solution</summary>


## Fix

```sh
DIR=/tmp/kubelings-kubeconfig
kubectl config use-context staging --kubeconfig $DIR/main
KUBECONFIG=$DIR/main:$DIR/extra kubectl config view --flatten > $DIR/merged
kubectl --kubeconfig $DIR/merged --context=staging auth can-i list pods
```

## The mechanics worth remembering

- **Merge rules**: `KUBECONFIG` is a colon-separated list; entries are merged
  left-to-right, first occurrence of a name wins, `current-context` comes
  from the first file that sets one. `--flatten` inlines credentials so the
  result is portable.
- `use-context` edits the *file* (`current-context:` line) — it's
  per-kubeconfig state, not per-shell. Two terminals sharing a file share
  the switch.
- `--minify` is the inverse tool: current context only, for handing someone
  access to exactly one thing.
- Namespace in a context is a default, not a boundary — `-n` still overrides
  it; RBAC is the boundary (M6.1).

## Prevention / takeaway

- Never leave `current-context` pointing at prod. Point it at something
  harmless; make prod access *deliberate* (`--context=prod` typed by hand, a
  separate file, or a separate terminal profile with its own `KUBECONFIG`).
- Put the context in your prompt (kube-ps1 or similar) — the cheapest
  guardrail that has prevented real incidents.
- Rotate the file, not just the cluster: old kubeconfigs are standing
  credentials. `kubectl config delete-context` + `delete-user` when access
  ends.
- CI systems get minified, flattened, single-context files — never a copy of
  a human's multi-cluster config.

</details>
