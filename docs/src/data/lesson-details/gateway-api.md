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
