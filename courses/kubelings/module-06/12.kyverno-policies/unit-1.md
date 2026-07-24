---
kind: unit
title: "Kyverno: the policy that blocked kube-system"
name: kyverno-policies-unit
---


## The situation

Someone shipped a "require app labels" policy this morning. Every other
team's deploys started bouncing within minutes:

```sh
kubectl -n default run test --image=registry.k8s.io/pause:3.9 --restart=Never --dry-run=server
# error: admission webhook "validate.kyverno.svc-fail" denied the request:
# ... require-app-label: 'every pod must carry an app label'
```

The policy:

```sh
kubectl get clusterpolicy pod-guardrails -o yaml
```

- `validationFailureAction: Enforce` — violations are hard denies
- `match: any: resources: kinds: [Pod]` — **no namespace scoping at all**

Kyverno (unlike Gatekeeper's Rego) is YAML-native: policies are `match` /
`exclude` blocks plus one of `validate`, `mutate`, or `generate`. This one
matches every Pod in the cluster — every team, every namespace. It's M6.3's
webhook-outage incident wearing a policy-engine costume: an admission
control with cluster-wide blast radius.

One thing saved the control plane itself: Kyverno's install ships default
**resourceFilters** (in the `kyverno` ConfigMap) that skip `kube-system`,
`kube-public`, and Kyverno's own namespace before policies even run. A
seatbelt you did not put on and can accidentally remove — never *rely* on it
for scoping; write the scope into the policy.

## Your task

Replace `pod-guardrails` with a version that:

1. **Scopes to `kubelings`** — other teams' namespaces (and the system ones)
   are out of bounds for this app policy. An `exclude` block is the
   alternative when a policy genuinely must be cluster-wide.
2. **Mutates instead of rejecting** for the label: a `mutate` rule with
   `+(app): unlabeled` adds the label only when missing — pods get fixed on
   the way in, nobody gets paged.
3. **Keeps one hard deny**: a `validate` rule rejecting `:latest` images in
   `kubelings`.

Then prove all three:

```sh
kubectl -n default run p1 --image=registry.k8s.io/pause:3.9 --restart=Never --dry-run=server                 # admitted
kubectl -n kubelings run p2 --image=nginx:1.25-alpine --restart=Never --dry-run=server -o jsonpath='{.metadata.labels.app}'  # label injected
kubectl -n kubelings run p3 --image=nginx:latest --restart=Never --dry-run=server                            # denied
```

<details>
<summary>Hint</summary>

`+(app):` is Kyverno's *add-if-absent* anchor — it never overwrites an
existing label. Full working policy in the Solution. Mutation shows up even
in `--dry-run=server` output, which makes testing free.

</details>

::simple-task
---
:tasks: tasks
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
kubectl apply -f - <<'EOF'
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: pod-guardrails
spec:
  validationFailureAction: Enforce
  background: false
  rules:
    - name: add-default-app-label
      match:
        any:
          - resources:
              kinds: ["Pod"]
              namespaces: ["kubelings"]
      mutate:
        patchStrategicMerge:
          metadata:
            labels:
              +(app): unlabeled
    - name: deny-latest-tag
      match:
        any:
          - resources:
              kinds: ["Pod"]
              namespaces: ["kubelings"]
      validate:
        message: "images must pin a tag — :latest is a mutable pointer"
        pattern:
          spec:
            containers:
              - image: "!*:latest"
EOF
```

Scoping via `namespaces:` in the match is the cleanest form here; the
equivalent `exclude` block (`exclude: any: resources: namespaces:
[kube-system, kyverno]`) is the right tool when the policy genuinely must be
cluster-wide.

## Why mutate > validate for labels

A missing label is a *fixable* defect — rejecting it converts a hygiene rule
into deploy friction and 2 a.m. pages. Mutation enforces the invariant
silently; validation is for things that must never exist (`:latest`,
privileged containers). Kyverno runs mutation before validation, so the
injected label also satisfies any label-validating rule.

## Prevention / takeaway

- **Every Enforce policy states its scope explicitly** — namespace match or
  exclude list, written in the policy. Kyverno's default resourceFilters
  shield `kube-system` and its own namespace, but that's install config, not
  policy: an edited ConfigMap away from the engine rejecting its own webhook
  pods.
- Roll out with `validationFailureAction: Audit` first; flip to `Enforce`
  after `kubectl get policyreports -A` comes back quiet.
- Kyverno's webhooks default fail-open for exactly the Jetstack (M6.3)
  reason; check `webhooks[].failurePolicy` before assuming otherwise.
- Same guardrail, two engines: this lesson's `deny-latest` is Gatekeeper's
  (M6.11) in YAML instead of Rego. Pick one engine per cluster; two admission
  layers double the ways a deploy can fail mysteriously.

</details>
