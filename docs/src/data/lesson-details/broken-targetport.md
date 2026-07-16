## The situation

You've internalized Module 1: *"service down, pods fine → check endpoints."* So
you check:

```sh
kubectl -n kubelings get endpoints search
```

```
NAME     ENDPOINTS                         AGE
search   10.244.1.7:8080,10.244.2.9:8080   5m
```

Populated! Selector matches, two pod IPs. And yet:

```
wget: can't connect to remote host (10.96.142.31): Connection refused
```

Look closer at those endpoints: `:8080`. Now ask nginx what it thinks:

```sh
kubectl -n kubelings exec deploy/search -- netstat -tlnp 2>/dev/null | grep LISTEN
```

```
tcp  0  0 0.0.0.0:80  0.0.0.0:*  LISTEN  1/nginx
```

The pod listens on **80**. The Service forwards to **8080**. Traffic arrives at
the right pod, knocks on a door where no process listens, and the kernel
answers with RST: *connection refused*.

Learn the distinction — it's half of network debugging:

- **refused** → you *reached* the host; nothing listens on that port. Wiring
  problem: targetPort, containerPort, process config.
- **timeout** → packets vanish. Reachability problem: NetworkPolicy, routing,
  wrong IP, firewall.

## Your task

Fix the chain so `http://search.kubelings.svc/` answers:

```
Service.port (80) → Service.targetPort (?) → containerPort → process
```

```sh
kubectl -n kubelings get svc search -o yaml | grep -A4 'ports:'
kubectl -n kubelings get deploy search -o jsonpath='{.spec.template.spec.containers[0].ports}'
```
