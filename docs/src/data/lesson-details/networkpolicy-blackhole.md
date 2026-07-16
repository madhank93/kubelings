## The situation

Friday's security PR was titled *"enable default-deny ingress (allow rules in
follow-up)"*. It merged. The follow-up didn't.

By Monday, `storefront` can't reach `orders-api`. DNS resolves, endpoints are
populated (you know to check that now), the pods are healthy — but connections
time out. There's no error in any log, because **NetworkPolicy drops packets
silently**. No RST, no ICMP, no event. Traffic just… stops existing.

```sh
kubectl -n kubelings get networkpolicy
```

```
NAME                   POD-SELECTOR   AGE
default-deny-ingress   <none>         3d
```

`podSelector: {}` selects **every pod in the namespace**. Once *any* policy
selects a pod for a direction (Ingress here), that pod flips from
"allow-everything" to **"deny everything except what some policy explicitly
allows."** With zero allow rules, that's a namespace-wide ingress blackhole.

> **Local note:** kind's default CNI (kindnet) doesn't *enforce* NetworkPolicy —
> the objects exist but traffic flows anyway. The check validates your policy
> objects; on iximiuz Labs (enforcing CNI) the same objects gate real packets.
> The reasoning you practice is identical.

## Your task

Keep the wall, add a door:

1. Don't delete `default-deny-ingress` — security was right to want it.
2. Write an *additional* policy allowing ingress to `orders-api` pods
   (`app=orders-api, tier=backend`) from frontend pods (`tier=frontend`) on
   port 80.

```sh
kubectl -n kubelings get pods --show-labels
kubectl -n kubelings describe networkpolicy default-deny-ingress
```
