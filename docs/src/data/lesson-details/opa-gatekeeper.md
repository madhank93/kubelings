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
