---
kind: unit
title: "Egress lockdown: the miner needs a phone line"
name: egress-lockdown-unit
---


## The situation

Re-read the cryptominer postmortem (lesson 6.2) with one question: *what did
the malware actually need from the network?* Not inbound — nobody connected
**to** it. It needed **outbound**: DNS to resolve the mining pool, then a
TCP session out. Same for every crypto-locker fetching keys, every backdoor
phoning home, every data exfil. **Compromise arrives however it arrives;
value leaves via egress.**

Now look at the payments namespace:

```sh
kubectl -n kubelings get networkpolicy
```

```
No resources found
```

No policies means **allow-everything, both directions**. Module 4's
NetworkPolicy lesson (4.2) built the ingress side of the fence. The SOC
mandate after 6.2 is the egress side: payments pods may talk to (a) DNS and
(b) the payment gateway. Nothing else, anywhere, ever.

## Your task

Three policies — the same default-deny-then-allow pattern as 4.2, pointed
outbound. Names matter (verify checks them):

**1. `egress-default-deny`** — the floor. Empty `podSelector` (everyone),
`policyTypes: [Egress]`, and **no egress rules at all** — a pod selected by
an Egress policy may only do what some policy's rules allow, and this one
allows nothing:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: {name: egress-default-deny, namespace: kubelings}
spec:
  podSelector: {}
  policyTypes: [Egress]
```

**2. `allow-dns`** — the allow everyone forgets, discovered as "the policy
works but *everything* broke": with egress denied, pods can't even resolve
`gateway.kubelings.svc`. Open 53/UDP + 53/TCP to kube-dns:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: {name: allow-dns, namespace: kubelings}
spec:
  podSelector: {}
  policyTypes: [Egress]
  egress:
    - to:
        - namespaceSelector:
            matchLabels: {kubernetes.io/metadata.name: kube-system}
      ports:
        - {port: 53, protocol: UDP}
        - {port: 53, protocol: TCP}
```

**3. `allow-payments-to-gateway`** — the one business flow, scoped tight:
`podSelector` app=payments (not everyone!), egress `to` pods labeled
app=gateway, port 80.

<details>
<summary>Hint</summary>

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: {name: allow-payments-to-gateway, namespace: kubelings}
spec:
  podSelector:
    matchLabels: {app: payments}
  policyTypes: [Egress]
  egress:
    - to:
        - podSelector:
            matchLabels: {app: gateway}
      ports:
        - {port: 80, protocol: TCP}
```

Remember 4.2's rule of composition: policies are **purely additive** — a pod's
allowed traffic is the union of every policy selecting it. Deny lives in one
place (selection with no matching rule); there is no "deny rule" to order or
conflict.

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


## Why egress is the high-value direction

Rank what each fence stops:

| | ingress deny (4.2) | egress deny (this) |
|---|---|---|
| lateral movement between namespaces | ✅ | ✅ (they can't reach out either) |
| C2 / phone-home | — | ✅ |
| data exfiltration | — | ✅ |
| cryptominer pool connection (6.2) | — | ✅ |
| supply-chain payload fetching its stage 2 | — | ✅ |

Most orgs deploy ingress policies first because *apps break loudly* without
them. Egress policies break nothing visible on day one — they only matter on
the day someone's inside. That's why they're the CKS fixation and the audit
finding: absence is invisible until it's the headline.

## The three egress gotchas

1. **DNS, always DNS.** Every egress lockdown that "broke everything" forgot
   port 53. Symptoms are maddeningly indirect — timeouts, not refusals —
   because resolution fails before connection starts (Zalando's lesson, 4.1:
   DNS is load-bearing).
2. **Enforcement is the CNI's job.** The API server stores NetworkPolicy
   objects; the CNI *implements* them (M8.6: the CNI is routes and rules on
   nodes). A CNI without policy support — including kind's default kindnet —
   accepts your YAML and enforces nothing. On this cluster the objects are
   the exercise; on the iximiuz playground and any Calico/Cilium cluster the
   same objects bite. **Always test policies on a CNI that enforces them** —
   `kubectl exec` a pod and try a connection that should now fail.
3. **In-cluster egress ≠ world egress.** `to: podSelector/namespaceSelector`
   governs cluster traffic; blocking the *internet* needs `ipBlock` (e.g.
   allow `10.0.0.0/8`, nothing else) — and NAT quirks per CNI make external
   filtering the least portable part of NetworkPolicy. For real
   allow-by-hostname egress (api.stripe.com only), you graduate to a
   CNI-level or mesh-level egress gateway.

## Prevention

- Namespace template = deny-all ingress **and** egress + allow-dns, from
  birth (pairs with the 6.5 automount-off default — the namespace starts
  safe and workloads *ask* for what they need).
- Every allow policy names one flow with one owner — `allow-payments-to-gateway`
  is reviewable; `allow-misc` is a hole with a YAML file.
- Alert on namespaces with zero NetworkPolicies (the fence that never got
  built) — one `kubectl get netpol -A` in your compliance script.

</details>
