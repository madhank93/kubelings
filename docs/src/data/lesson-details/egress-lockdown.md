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
