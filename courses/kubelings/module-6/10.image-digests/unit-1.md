---
kind: unit
title: "Tags lie, digests don't: pin the supply chain"
name: image-digests-unit
---


## The situation

Security review question 7: *"How do you know the image running in prod is
the image you audited?"* The deployment says:

```yaml
image: nginx:1.27-alpine
```

That's not an answer — it's a **mutable pointer**. A tag is a name the
registry maps to an image *right now*. The mapping changes when the publisher
pushes a rebuild to the same tag (routine!), or when an attacker with
registry access re-points it (the supply-chain attack in one move). Two nodes
pulling the "same" tag on different days can run different code. Your audit
covered a snapshot of where the pointer aimed; the pointer moved on.

A **digest** is different in kind, not degree: `sha256:...` is computed from
the image content itself. `nginx@sha256:abc...` names *bytes*, not a mapping.
The registry can't re-point it; a tampered image can't match it; the runtime
verifies it on pull. Content-addressing — the same trick git uses.

## Your task

Pin `checkout` to the digest of what it's *verifiably running right now*.

1. Kubernetes already resolved the tag when it pulled — the pod's status
   records the truth:

```sh
kubectl -n kubelings get pods -l app=checkout \
  -o jsonpath='{.items[0].status.containerStatuses[0].imageID}{"\n"}'
```

```
docker.io/library/nginx@sha256:9c8f...
```

Note the two fields side by side: `.image` is what the spec *asked for* (the
tag); `.imageID` is what the node *got* (the digest). This lesson closes that
gap.

2. Patch the deployment's image to `nginx@sha256:<that digest>` and let it
   roll.

<details>
<summary>Hint</summary>

```sh
DIGEST=$(kubectl -n kubelings get pods -l app=checkout \
  -o jsonpath='{.items[0].status.containerStatuses[0].imageID}' | sed 's/.*@//')
kubectl -n kubelings set image deploy/checkout checkout="nginx@${DIGEST}"
kubectl -n kubelings rollout status deploy/checkout
```

Tag and digest can't be combined meaningfully — `nginx:1.27-alpine@sha256:...`
is legal syntax but the tag part is *ignored*, kept only as a comment for
humans. The digest is the whole identity.

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


## What pinning buys (and what it doesn't)

| Threat | Tag | Digest |
|---|---|---|
| registry re-points tag (attack or rebuild) | runs the new thing silently | **immune** |
| node A and node B run different builds of "the same version" | happens | impossible |
| `:latest`-style drift (your 1.6 lesson) | happens | impossible |
| image *contains* a vulnerability | vulnerable | **equally vulnerable** |

Read the last row twice: a digest freezes *identity*, not *quality*. You're
now perfectly, reproducibly running whatever was in that image — including
its CVEs, forever, until you consciously move the pin. Which is the honest
trade: **digests convert "the registry changed something under us" into "we
must ship digest bumps ourselves."** That's what bots (Renovate/Dependabot)
are for: they watch the tag, open a PR moving the digest, CI scans it, a
human merges. Update flow with review, instead of update flow by surprise.

## The supply-chain ladder (from lesson 6.8, now with rung one done)

1. **Pin by digest** ← you are here. Free, immediate, kubectl-only.
2. **Scan continuously**: trivy/grype in CI *and* re-scan the registry on a
   schedule — new CVEs land in old images. (Runs fine as a cluster Job, same
   pattern as kube-bench in 6.7, but needs its vulnerability DB downloaded —
   try it on the playground: `trivy image nginx@<your digest>`.)
3. **Sign & verify**: cosign in CI, an admission webhook (6.3's machinery)
   refusing unsigned digests. At that rung, even a fully compromised
   registry can't get code into your cluster — the trust root moved to your
   CI's keys.

And the admission-control preview: policy engines (Kyverno, Gatekeeper)
express "no `:latest`", "digests only in prod", "signed images only" as
cluster policy — the same webhook seat Pod Security occupies (6.4), pointed
at provenance.

## Prevention

- CI renders manifests (3.6) → a ~5-line check rejects `image:` without
  `@sha256:` in prod overlays. Cheapest control in this module.
- Keep the human-readable tag *in a comment or annotation* next to the pin —
  future-you wants to know the digest is "1.27-alpine, pulled 2026-07-07".
- `imagePullPolicy: IfNotPresent` + digest = optimal: identical content
  cached forever, no daily re-pull races. (`Always` + tag, the old advice,
  was compensating for exactly the mutability you just removed.)

</details>
