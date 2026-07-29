> **Drill** — this is a synthetic composite of a failure mode reported across
> many production clusters (cloud load balancers in front of `Local` Services,
> node drains, single-node placement), not a specific company's incident.

## The situation

`checkout` needs the **real client IP** — fraud scoring keys off it, and legal
wants it in the access log. So the Service was created with
`externalTrafficPolicy: Local`, which is the correct answer to that requirement:
traffic that lands on a node is delivered to a pod **on that same node**, with no
second hop and no SNAT, so the source address survives.

Then someone pinned all three replicas to one node.

```sh
kubectl -n kubelings get pods -o wide -l app=checkout   # NODE column: all the same
kubectl -n kubelings get svc checkout -o jsonpath='{.spec.externalTrafficPolicy}{"\n"}'
```

From one node, `:30090` answers. From every other node, it doesn't answer at
all — no reset you can attribute, no 502 from an ingress, no pod log. In a real
cluster the load balancer is what discovers this, and only if its health checks
are pointed at the right port. Otherwise it keeps sending a share of production
traffic to nodes that will silently drop it.

```sh
for ip in $(kubectl get nodes -l '!node-role.kubernetes.io/control-plane' \
              -o jsonpath='{range .items[*]}{.status.addresses[?(@.type=="InternalIP")].address}{" "}{end}'); do
  echo -n "$ip: "; curl -fsS -m 3 "http://$ip:30090/" >/dev/null 2>&1 && echo ok || echo BLACKHOLE
done
```

> Probe from a node that isn't the target: traffic a node generates *itself*
> reaches a NodePort under the cluster policy, so a node always appears healthy
> to itself even when it blackholes everyone else's packets.

<figure class="lesson-diagram">
<img src="/diagrams/pattern-etp-blackhole-local.svg" alt="pattern-etp-blackhole local diagram" loading="lazy">
</figure>

## Your task

Keep the client IP. Lose the blackhole:

1. `externalTrafficPolicy` stays **`Local`** — SNAT-ing it away with `Cluster` is
   the fix that breaks the requirement.
2. The Service stays a NodePort on **30090**.
3. **Every worker node** must answer on `:30090` — i.e. every node must host a
   ready `checkout` endpoint.
4. `checkout` keeps at least 2 available replicas.
