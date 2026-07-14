---
kind: unit
title: "Capsule: the tenant that hit its walls"
name: multi-tenancy-capsule-unit
---


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

<details>
<summary>Hint</summary>

Watch what Capsule did the moment alice's namespace landed:

```sh
kubectl -n alpha-dev get rolebindings
# capsule-team-alpha-0-admin … alice
```

The RoleBinding, the tenant label, quota accounting — all injected by the
mutating webhook at namespace creation. That's why the namespace must be
created *by the owner*: an admin-created namespace belongs to no tenant.

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
kubectl patch tenant team-alpha --type=merge \
  -p '{"spec":{"namespaceOptions":{"quota":3}}}'
kubectl create ns alpha-dev --as=alice --as-group=projectcapsule.dev
```

## What a Tenant actually buys you

Everything M6 taught you to assemble by hand, stamped per-namespace by a
controller:

- **RBAC**: owners get admin *inside* their namespaces, nothing outside —
  the RoleBindings appear on namespace creation (M6.1's least-privilege,
  automated).
- **Quotas & limits**: `namespaceOptions.quota` caps namespace count;
  `resourceQuotas` / `limitRanges` on the Tenant stamp M8.3-style quotas
  into every tenant namespace.
- **Guardrails**: allowed ingress classes, storage classes, registries
  (`imagePullPolicies`, allowed repos — M6.13's registry pinning as tenant
  policy), forbidden labels, pod security. All admission-enforced.

The mechanism is familiar by now: mutating + validating webhooks (M7.7)
watching namespace and workload writes, keyed off tenant ownership. The
`--as-group=projectcapsule.dev` group is how Capsule knows a request comes
from a tenant user rather than a cluster admin — in production that group
arrives via your SSO/OIDC groups claim instead of impersonation.

## Prevention / takeaway

- Tenant quotas are *capacity promises*, and teams outgrow them — treat
  the "Cannot exceed" denial as a signal to review the tenant's sizing,
  not as an error to work around by having admins create namespaces
  outside the tenant (that orphans them from every wall).
- The isolation test is two `auth can-i` lines — run them from CI per
  tenant; isolation that isn't tested erodes (same discipline as M6.11's
  "test the deny").
- One cluster + tenants vs many clusters is a real architecture decision:
  tenants share a control plane, node pool, and blast radius (M9's
  cascades hit every tenant at once). Capsule is the cheap end; Cluster
  API (next lesson) is the expensive end.

</details>
