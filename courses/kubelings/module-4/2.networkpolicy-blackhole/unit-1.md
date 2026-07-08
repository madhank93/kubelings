---
kind: unit
title: "NetworkPolicy blackhole: default-deny ate my traffic"
name: networkpolicy-blackhole-unit
---


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

<details>
<summary>Hint</summary>

Policies are additive — a second policy opens what the first denies:

```sh
kubectl apply -n kubelings -f - <<'EOF'
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-frontend-to-orders
spec:
  podSelector:
    matchLabels: {app: orders-api}
  policyTypes: [Ingress]
  ingress:
    - from:
        - podSelector:
            matchLabels: {tier: frontend}
      ports:
        - {protocol: TCP, port: 80}
EOF
```

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


## The mental model

NetworkPolicy is **allowlist-only**. There is no "deny rule" — there is only:

1. **Unselected pod** → all traffic allowed (the scary default).
2. **Selected pod** → all denied in that direction, except unions of allows.

Policies never conflict; they only add doors. `default-deny` works by selecting
everything and allowing nothing — then each app team ships its own door.

## Fix

The `allow-frontend-to-orders` policy from the hint. Read it as a sentence:
*"pods labeled app=orders-api accept ingress on TCP/80 from pods labeled
tier=frontend, in this namespace."*

Cross-namespace sources need `namespaceSelector` (and remember: an empty `from:`
allows *everywhere*, which is rarely what security meant).

## Debugging silent drops

No logs, so triage by elimination — this order:

```sh
kubectl -n <ns> get endpoints <svc>        # rule out selector mismatch (M1!)
kubectl -n <ns> get networkpolicy         # anything selecting the target?
kubectl -n <ns> describe netpol <name>    # what does it actually allow?
```

Timeout (not refused) + healthy endpoints + a policy in the namespace ≈ 90%
it's the policy.

## Prevention

- Default-deny and its allow rules ship **in the same PR**. Non-negotiable.
- Don't forget DNS when you add `policyTypes: [Egress]` — blocking UDP/53 to
  kube-dns is the classic self-own (it starved half the incidents in the
  [Incident Library](https://kubelings.madhan.app/reference/incident-library/)).
- Test policies with a throwaway labeled pod, exactly like the verify check does.

</details>
