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
