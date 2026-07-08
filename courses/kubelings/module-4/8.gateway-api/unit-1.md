---
kind: unit
title: "Gateway API: routing with a role for everyone"
name: gateway-api-unit
---


## The situation

Your Ingress fix (lesson 4.7) worked, but the retro surfaced the deeper
problem: **one object, three owners.** TLS config, load-balancer class, and
app routes all live in the same Ingress resource — so the platform team, the
security team, and every app team keep editing the same YAML, and annotations
(`nginx.ingress.kubernetes.io/...`) carry everything the spec can't say.

**Gateway API** is the successor, and its core idea is org-chart-shaped:
split the object along the ownership seams.

```
GatewayClass   "which implementation"        owner: infra provider
    ▲
Gateway        "a listener: port, TLS, who   owner: platform team
    ▲           may attach"
HTTPRoute      "match these requests →       owner: app team
                this Service"
```

The platform half is already deployed. Look at it:

```sh
kubectl api-resources | grep gateway.networking   # new kinds — CRDs! (M7.5)
kubectl get gatewayclass
kubectl -n kubelings get gateway web -o yaml | grep -B2 -A6 listeners
```

Note `allowedRoutes: namespaces: from: Same` on the listener — attachment is
**consent-based**. An HTTPRoute doesn't just claim a Gateway (the way any
Ingress could claim any class); the Gateway declares who may attach. That
handshake is the security fix Ingress never had.

## Your task

Write the app team's half: an HTTPRoute named `catalog` in `kubelings` that

- attaches to the `web` Gateway (`parentRefs`),
- matches path prefix `/catalog`,
- sends traffic to the `catalog` Service on port 80 (`backendRefs`).

Every ref is a name that must agree with something real — the 4.7 chain, new
vocabulary.

<details>
<summary>Hint</summary>

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: catalog
  namespace: kubelings
spec:
  parentRefs:
    - name: web
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /catalog
      backendRefs:
        - name: catalog
          port: 80
```

`kubectl -n kubelings describe httproute catalog` — the `Parents` status
section is where a controller would report Accepted/ResolvedRefs conditions
per attachment.

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


## Ingress → Gateway API, translated

| Ingress concept | Gateway API | What improved |
|---|---|---|
| `ingressClassName` | GatewayClass | explicit contract, versioned CRD |
| the LB/listener config (annotations) | Gateway `listeners` | typed spec: port, protocol, TLS, hostname |
| `rules:` | HTTPRoute (+ GRPCRoute, TLSRoute, TCPRoute) | per-protocol kinds, richer matching |
| — (anyone claims any class) | `parentRefs` ⇄ `allowedRoutes` | **two-sided consent**; cross-namespace by policy, not accident |
| backend Service+port | `backendRefs` | plus weights — canary (2.13) as a first-class field |

That last row deserves a look — the blue/green lesson's selector flip,
expressed as routing config instead:

```yaml
backendRefs:
  - {name: catalog-v1, port: 80, weight: 90}
  - {name: catalog-v2, port: 80, weight: 10}
```

Per-*request* 90/10 (not per-connection like the Service-selector trick), and
shifting traffic is a one-field diff an app team can own.

## The status discipline

On this cluster no controller implements `kubelings.dev/unmanaged`, so your
route's `Parents` status stays empty — the same "object is the contract,
controller is fulfillment" split as 4.7, made visible. On a real cluster
(istio, envoy-gateway, cilium, nginx-gateway-fabric all ship Gateway API
implementations) debugging is condition-driven, top down:

```
Gateway:    Accepted? Programmed?          ← platform's pager
HTTPRoute:  Accepted per parentRef?        ← "may I attach" failed → allowedRoutes
            ResolvedRefs?                  ← backendRef name/port wrong → this lesson
Service:    endpoints non-empty?           ← 4.7's ladder, unchanged from here down
```

Statuses name the guilty layer *and therefore the team* — which was the whole
point of splitting the object.

## When to use which (2026 honest version)

- **Ingress** is everywhere, frozen (feature-complete, no evolution), and
  fine for "one team, one cluster, route HTTP to services."
- **Gateway API** is GA, is where all new routing features land (gRPC,
  traffic splitting, cross-namespace, mesh via GAMMA), and is the right
  default for platforms with more than one team. Migration is mechanical
  (the table above); the mental model you carry over unchanged is the one
  this module drilled: **a chain of names, each of which must agree.**

</details>
