---
kind: unit
title: "Ingress: three names that must agree"
name: ingress-wiring-unit
---


## The situation

The frontend team shipped an Ingress for `shop.example.com`. The site 404s.
Their theory: "the ingress controller is broken." Your job: prove it's wiring.

An Ingress is L7 routing *declared as data*: an ingress **controller** (nginx,
traefik, HAProxy, a cloud LB) watches Ingress objects and reprograms itself to
match. The object itself is nothing but a chain of names:

```
host ──▶ path ──▶ backend service NAME ──▶ service PORT ──▶ endpoints ──▶ pods
```

Every arrow is a lookup that can miss — and a miss doesn't error at apply
time. `kubectl apply` validated the *syntax* happily; the names are only
resolved when traffic (or the controller) tries to follow them. Sound
familiar? It's lesson 1.4's selector mismatch and lesson 4.3's targetPort bug,
one layer up. Walk the chain:

```sh
kubectl -n kubelings describe ingress shop        # note "<error: service ... not found>"
kubectl -n kubelings get svc                      # what actually exists?
kubectl -n kubelings get svc storefront -o yaml | grep -A5 ports
```

Two breaks, same chain:

1. Backend name: `store-front` — the Service is `storefront`. One hyphen.
2. Backend port: `80` — the Service exposes port `8080` (named `web`). The
   pod listens on 80, but the Ingress talks to the *Service*, not the pod;
   the Service's `targetPort` handles the last hop.

## Your task

Fix the Ingress (leave Deployment and Service alone — they're what the rest
of the cluster already depends on):

- backend service name → the one that exists,
- backend port → one the Service actually exposes (by number **or** by name;
  named ports survive renumbering — the same argument as lesson 4.3).

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings patch ingress shop --type=json -p '[
  {"op":"replace","path":"/spec/rules/0/http/paths/0/backend/service/name","value":"storefront"},
  {"op":"replace","path":"/spec/rules/0/http/paths/0/backend/service/port","value":{"name":"web"}}]'
```

`kubectl describe ingress shop` again — the backend column now resolves to
real endpoint IPs instead of an error.

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


## The debugging ladder for "my ingress 404s"

Top of the chain to the bottom — each step is a lesson you've done:

| Hop | Check | Classic failure |
|---|---|---|
| DNS → controller | does the hostname resolve to the LB at all? | DNS not pointed, wrong LB |
| controller → rule | `describe ingress`: host/path match the request? | `pathType: Exact` vs trailing slash; wrong `ingressClassName` so **no controller claims it** |
| rule → service | backend name exists in the **same namespace**? | typo (this lesson) — Ingress can't cross namespaces |
| service → port | port number/name exists on the Service? | this lesson |
| service → pods | `get endpoints` non-empty? | selector mismatch (1.4), blackhole (4.2/4.3) |
| pods → 200 | `kubectl exec` + curl the pod directly | app itself; probes lying (2.8) |

Note the one field this lesson's cluster let us skip: **`ingressClassName`**.
Real clusters run one or more controllers, and an Ingress that doesn't name
its class may be claimed by nobody — syntactically perfect, functionally
invisible. It's the first thing to check when `describe` shows *no* events.

On this kind cluster no controller is installed, so the object-level wiring is
the whole exercise — which is honest to the layer split: **the object is the
contract; the controller is fulfillment**. The chain you verified is exactly
what any controller will program.

## Where this is going: Gateway API

Ingress is ubiquitous but cramped — one resource carries infra concerns (TLS,
class) *and* app concerns (routes), so teams collide on it. Its successor,
**Gateway API**, splits the roles: `GatewayClass` (which implementation) →
`Gateway` (a listener, owned by platform) → `HTTPRoute` (routes, owned by app
teams, attached by reference). Same name-chain discipline, more hops:
route `parentRefs` must name a Gateway that allows the route's namespace, and
`backendRefs` must name a Service+port — the exact two things you just fixed.
Debug it with the same ladder.

## Prevention

- Named service ports in Ingress backends — renumbering a Service then breaks
  nothing.
- CI check: for every Ingress backend, assert the Service+port exists in the
  namespace (a ~10-line script against the rendered manifests — the apply
  won't do it for you).
- Alert on Ingress objects with no matching class/controller events and on
  backend services with zero endpoints: both are "deployed but dark."

</details>
