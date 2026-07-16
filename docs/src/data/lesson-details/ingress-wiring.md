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
