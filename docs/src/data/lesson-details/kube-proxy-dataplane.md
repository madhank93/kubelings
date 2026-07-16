> **Reading.** Inspecting iptables needs node access (outside the kubectl
> sandbox), so this is a guided tour with the exact commands to run when you
> *do* have a node (any kind cluster on your own machine:
> `docker exec -it <node> iptables -t nat -L KUBE-SERVICES | head`).
> Cited alongside: Datadog's KubeCon talk
> ["10 ways to shoot yourself in the foot"](https://www.youtube.com/watch?v=QKI-JRs2RIE),
> which is this lesson's failure modes told from production.

## The mystery you've been stepping around

Every Service lesson in this module used a ClusterIP. Now try to find one:

```sh
kubectl -n kubelings get svc            # say, 10.96.143.7
kubectl -n kubelings exec deploy/web -- ip addr   # ...not here
```

No pod, no node, no interface anywhere owns that IP. Ping it — usually dead.
Yet TCP to it works from every pod. **A ClusterIP is not a destination; it's
a rewrite rule.** And despite the name, kube-proxy (in its default mode)
proxies nothing — no process ever touches your packets. kube-proxy is a
*controller* (7.1's pattern, wearing a network hat): it watches Services and
EndpointSlices (7.8's list+watch) and programs the **kernel** to do the work.

## The packet's actual journey (iptables mode)

Pod sends to `10.96.143.7:80`. Before routing, the kernel walks netfilter's
NAT table, where kube-proxy has written:

```
KUBE-SERVICES                    match dst 10.96.143.7:80
  → KUBE-SVC-XYZ                 the "load balancer": N rules with
                                 probability 1/N, (N-1)/N-of-rest, ... 1
  → KUBE-SEP-ABC                 one endpoint: DNAT to 10.244.1.17:8080
```

Three things fall out of reading that chain:

1. **Load balancing is a dice roll at connection setup.** Random per
   *connection*, not per request, not least-loaded — which is exactly why the
   blue/green canary ratio (2.13) is approximate and long-lived connections
   pin to one backend.
2. **The port rewrite is the DNAT** — `port: 80 → targetPort: 8080` from
   lesson 4.3 is literally this line. A wrong targetPort is a DNAT to a
   closed port; now you know *where* the connection refused comes from.
3. **conntrack remembers the roll** (4.5). The DNAT decision is stored per
   connection so reply packets get un-rewritten. Every ClusterIP flow =
   one conntrack entry — the table the loveholidays/Preply incidents filled.
   And when an endpoint dies, its established entries can linger, black-
   holing traffic — the graceful-shutdown race (4.6) has a kernel layer too.

An empty Service — no endpoints (1.4, 4.2) — has no KUBE-SEP chains at all;
the packet falls through to a REJECT. "Connection refused from a blackhole"
is that rule.

## The three backends (and why yours matters at scale)

| mode | mechanism | lookup cost | update cost | notes |
|---|---|---|---|---|
| **iptables** (default) | rule chains | O(#services) per first packet | full-table rewrites | fine to ~thousands of services, then both costs bite |
| **IPVS** | in-kernel hash-table LB | O(1) | incremental | real LB algorithms (rr, lc); the classic big-cluster fix — with its own edge cases (Datadog's talk has scars) |
| **nftables** (newest) | nftables maps | ~O(1) | incremental | iptables' successor; becoming the modern default |

Check any cluster's mode: `kubectl -n kube-system get cm kube-proxy -o yaml | grep mode`
(empty = iptables). And the increasingly common fourth answer: **no
kube-proxy at all** — Cilium/eBPF replaces the whole layer (the same Cilium
whose routes an OS update deleted in 8.6 — every dataplane choice is also a
failure-domain choice).

## Odds and ends that bite

- **hostNetwork pods** (DaemonSets, 2.2) share the node's network namespace —
  ClusterIPs resolve via the *node's* rules, and binding node ports can
  collide with NodePorts (4.4).
- **NodePort is the same trick** — one more match rule (`dst-port 30080`)
  in front of the same KUBE-SVC chain.
- `externalTrafficPolicy: Local` skips the SNAT that hides client IPs — the
  "why does my app see the node's IP?" question, answered in one field.
- Debug ladder when "the Service is broken": endpoints exist? (1.4) →
  targetPort right? (4.3) → policy allows? (4.2/6.9) → *then* node-level:
  conntrack counters (4.5) and the chains above. The kernel layer is last
  because it's least often guilty — but when it is, nothing above it will
  show you.

*No check — study, then advance.*
