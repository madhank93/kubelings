---
kind: unit
title: "Connection refused: port vs targetPort"
name: broken-targetport-unit
---


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

<details>
<summary>Hint</summary>

```sh
kubectl -n kubelings patch svc search --type=json -p '[
  {"op":"replace","path":"/spec/ports/0/targetPort","value":80}
]'
```

Endpoints re-render with `:80` within seconds — no pod restart, the Service is
just a routing rule.

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


## Root cause

`targetPort: 8080` on a Service whose pods serve on 80. Endpoints happily
listed pod IPs *with the wrong port* — the endpoints controller checks label
match and readiness, **not** whether anything listens on targetPort. "Endpoints
populated" is necessary, not sufficient.

## Fix

```sh
kubectl -n kubelings patch svc search --type=json -p '[
  {"op":"replace","path":"/spec/ports/0/targetPort","value":80}
]'
```

## The port glossary (tattoo this somewhere)

| Field | Lives on | Means |
|---|---|---|
| `port` | Service | what clients dial on the Service VIP/DNS |
| `targetPort` | Service | which pod port traffic forwards to |
| `containerPort` | Pod | documentation of what the process listens on |
| `nodePort` | Service (type NodePort) | port opened on every node (next lesson) |

Trap: `containerPort` is **not enforced** — it's metadata. The process's own
config decides what it binds. When in doubt, ask the pod (`netstat`/`ss`), not
the YAML.

Robust pattern: **named ports.** Container declares `ports: [{name: http,
containerPort: 80}]`, Service says `targetPort: http` — rename-proof and
self-documenting.

## Prevention

- Smoke-test through the Service DNS after every Service change — a one-line
  `kubectl run … wget` (exactly what the verify does).
- Named ports in every chart you write from today.
- File the reflex: **refused = wrong door, timeout = no road.** It routes you to
  the right module instantly.

</details>
