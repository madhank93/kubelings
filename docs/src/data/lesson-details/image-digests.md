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
