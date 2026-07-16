## The situation

Two teams, one cluster. Instead of a cluster per team (expensive) or a
pile of hand-rolled RBAC + quotas + NetworkPolicies per namespace
(unmaintainable — you built exactly that pile through M6), **Capsule**
adds one abstraction: the **Tenant**. A tenant owns a *set* of namespaces;
its owners self-serve namespaces inside their walls, and Capsule's
admission webhooks enforce the walls.

```sh
kubectl get tenants
# NAME         STATE    NAMESPACE QUOTA   NAMESPACE COUNT
# team-alpha   Active   1                 1
# team-beta    Active                     1
```

Alice (team-alpha's owner) needs a namespace for a new service. Owners act
through Capsule's user group — on this playground you impersonate her:

```sh
kubectl create ns alpha-dev --as=alice --as-group=projectcapsule.dev
# Error from server (Forbidden): admission webhook
# "namespaces.validating.projectcapsule.dev" denied the request:
# Cannot exceed Namespace quota: please, reach out to the system administrators
```

`team-alpha` was sized back when the team ran one service:
`namespaceOptions.quota: 1` — and `alpha-legacy` already consumes it. The
denial isn't RBAC (alice is *allowed* to create namespaces — that's the
tenant promise); it's the tenant's own capacity wall. You are the system
administrator the error mentions.

## Your task

1. Resize the tenant:

   ```sh
   kubectl patch tenant team-alpha --type=merge \
     -p '{"spec":{"namespaceOptions":{"quota":3}}}'
   ```

2. Self-serve the namespace **as alice** — the point is that no admin
   creates it for her:

   ```sh
   kubectl create ns alpha-dev --as=alice --as-group=projectcapsule.dev
   ```

3. Prove the walls: alice is full owner inside, blind outside:

   ```sh
   kubectl auth can-i create pods -n alpha-dev --as=alice --as-group=projectcapsule.dev   # yes
   kubectl auth can-i get pods    -n beta-dev  --as=alice --as-group=projectcapsule.dev   # no
   kubectl get ns alpha-dev -o jsonpath='{.metadata.labels}'    # capsule.clastix.io/tenant: team-alpha
   ```
