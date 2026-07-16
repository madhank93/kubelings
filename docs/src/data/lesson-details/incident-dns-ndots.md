## The real incident

**January 7, 2019. Zalando — Europe's biggest fashion platform.** A downstream
service slows, the aggregation layer starts retrying, request volume spikes. Each
retried request opens fresh connections; each connection resolves the target's
hostname again (the Node.js apps had **no DNS caching**).

And then the multiplier kicks in. Every Kubernetes pod with the default
`dnsPolicy: ClusterFirst` gets a resolv.conf like this:

```
search kubelings.svc.cluster.local svc.cluster.local cluster.local ...
options ndots:5
```

`ndots:5` means: *any name with fewer than 5 dots is tried through every search
domain first.* `payments.example.com` has 2 dots — so the resolver tries
`payments.example.com.kubelings.svc.cluster.local`,
`.svc.cluster.local`, `.cluster.local`, … (each as A **and** AAAA) before finally
asking for the real name. **~10 DNS queries for every single external lookup.**

CoreDNS — running with a `100Mi` memory limit — ate the amplified flood, blew
past its limit, and was **OOMKilled. All replicas. Simultaneously.** Total
cluster DNS outage. The monitoring stack needed DNS too, so paging failed.
Fashion store error rates stayed high for **over an hour**.

Source: [Zalando postmortem — Jan 2019 DNS outage](https://github.com/zalando-incubator/kubernetes-on-aws/blob/dev/docs/postmortems/jan-2019-dns-outage.md)

## This cluster, right now

The `checkout` Deployment in `kubelings` resolves `payments.example.com` in a
loop — stand-in for prod retry traffic. It runs with stock DNS settings, i.e.
the amplifier is armed.

## Your task

Defuse the amplifier without losing cluster DNS:

1. Look at what the pod's resolver is actually configured to do.
2. Count the fan-out yourself (debug a lookup if you like).
3. Fix the Deployment so external lookups cost **one** query: keep
   `dnsPolicy: ClusterFirst`, but set `ndots` to `1` (or `2`) via
   `spec.template.spec.dnsConfig`.

```sh
POD=$(kubectl -n kubelings get pods -l app=checkout -o name | head -1)
kubectl -n kubelings exec ${POD#pod/} -- cat /etc/resolv.conf
```
