---
kind: unit
title: "Gatekeeper: the policy that rejected nothing"
name: opa-gatekeeper-unit
---


## The situation

Security mandated "no `:latest` tags in `kubelings`" months ago, and
Gatekeeper is dutifully installed:

```sh
kubectl -n gatekeeper-system get pods
kubectl get constrainttemplates
kubectl get k8sdenylatest
```

Template, Constraint, `enforcementAction: deny` — all present. Now watch it
work:

```sh
kubectl -n kubelings run should-fail --image=nginx:latest --restart=Never --dry-run=server
# pod/should-fail created (server dry run)
```

Admitted. The policy has been rejecting **nothing** since the day it shipped.

Gatekeeper's pipeline: a **ConstraintTemplate** carries Rego (OPA's policy
language) and declares a CRD kind; each **Constraint** (here `K8sDenyLatest`)
instantiates it against kinds/namespaces. On every admission request the
webhook evaluates `violation[...]` rules against `input.review.object` — the
AdmissionReview payload. If no `violation` fires, the request is admitted.

Read the shipped Rego:

```sh
kubectl get constrainttemplate k8sdenylatest -o jsonpath='{.spec.targets[0].rego}'
```

```rego
violation[{"msg": msg}] {
  image := input.review.object.image
  endswith(image, ":latest")
  ...
}
```

`input.review.object` is the **Pod**. Pods have no `.image` at the root —
images live at `.spec.containers[*].image`. In Rego, referencing a missing
field doesn't error; the expression is simply *undefined* and the rule body
silently never matches. A typo'd path equals an always-allow policy —
that's the sharpest edge in the language.

## Your task

Fix the Rego so the rule iterates the actual containers array:

```rego
violation[{"msg": msg}] {
  some c
  image := input.review.object.spec.containers[c].image
  endswith(image, ":latest")
  msg := sprintf("container image %v uses the :latest tag — pin a version", [image])
}
```

Apply it by editing the ConstraintTemplate (`kubectl edit constrainttemplate
k8sdenylatest`, or re-apply the full YAML). Then prove both directions:

```sh
kubectl -n kubelings run p1 --image=nginx:latest      --restart=Never --dry-run=server   # rejected
kubectl -n kubelings run p2 --image=nginx:1.25-alpine --restart=Never --dry-run=server   # admitted
```

<details>
<summary>Hint</summary>

After editing the template, Gatekeeper recompiles it — give it a few
seconds, then check for compile errors:

```sh
kubectl get constrainttemplate k8sdenylatest -o jsonpath='{.status.byPod[*].errors}'
```

Empty errors + both probes behaving = done. Also worth covering
`initContainers` with a second `violation` block in real clusters.

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


## Root cause

The Rego read `input.review.object.image` — a field that exists on nothing.
Undefined references in Rego don't raise errors; the rule body just never
succeeds, so no violation is ever produced and every request is admitted.
The policy *looked* deployed, returned no errors, and enforced nothing.

## Fix

```sh
kubectl apply -f - <<'EOF'
apiVersion: templates.gatekeeper.sh/v1
kind: ConstraintTemplate
metadata:
  name: k8sdenylatest
spec:
  crd:
    spec:
      names:
        kind: K8sDenyLatest
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
        package k8sdenylatest

        violation[{"msg": msg}] {
          some c
          image := input.review.object.spec.containers[c].image
          endswith(image, ":latest")
          msg := sprintf("container image %v uses the :latest tag — pin a version", [image])
        }
EOF
```

## Prevention / takeaway

- **Test policies in both directions** — a bad pod that must fail AND a good
  pod that must pass. An untested deny policy is usually an allow policy.
  `--dry-run=server` makes this a free CI check.
- Gatekeeper's **audit** controller catches this class in retrospect:
  `kubectl get k8sdenylatest deny-latest-tag -o jsonpath='{.status.totalViolations}'`
  — a deny policy with zero audit violations across a dirty cluster is a
  smell, not a success.
- `enforcementAction: dryrun` is the safe rollout mode: violations land in
  status without blocking anyone — flip to `deny` after audit looks right.
- The webhook's failure mode matters as much as its logic: Gatekeeper
  defaults to `failurePolicy: Ignore` (fail-open) exactly because a
  fail-closed policy webhook can take down the cluster's write path — that's
  M6.3's Jetstack incident, and the same tradeoff returns in the Kyverno
  lesson next door.
- Why `:latest` at all? It's a *mutable* pointer — M6.10 (`image-digests`)
  showed drift; the scanning lesson (M6.13) closes the loop with digests.

</details>
