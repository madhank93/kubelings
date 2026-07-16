## The real incident

**Ravelin**, fraud-detection platform, GKE. Their write-up opens with a
confession every platform team eventually makes: *deploys cause a blip of 502s,
and we'd been ignoring it.*

Source: [Kubernetes' dirty endpoint secret and Ingress — Phil Pearl / Ravelin](https://philpearl.github.io/post/k8s_ingress/)

The mental model everyone has: pod is removed from endpoints → traffic stops →
*then* the pod shuts down. Orderly. Sequential.

The truth Ravelin documented: **those two things happen in parallel.** When a
pod starts terminating:

1. The kubelet sends the container SIGTERM, **and, at the same time,**
2. the endpoints controller starts removing the pod — an update that must then
   propagate to *every* kube-proxy on every node, every ingress controller,
   every cloud LB health check.

Step 1 is one process on one node: milliseconds. Step 2 fans out across the
whole cluster: **seconds**. If the app exits promptly on SIGTERM — like every
well-behaved server — it's gone while nodes are *still routing new requests to
it*. Each one: 502.

The fix is gloriously dumb, and Ravelin says so: **don't die yet. Sleep.** A
preStop hook that waits a few seconds keeps serving while the routing world
catches up, then the app gets SIGTERM and exits clean.

## This cluster, right now

`checkout-api` has the anti-pattern turned up: no preStop, and
`terminationGracePeriodSeconds: 1` — the pod is SIGKILLed one second after
SIGTERM, mid-flight requests be damned. Every rollout is a micro-outage.

## Your task

Make termination outlast endpoint propagation:

1. Add a `preStop` hook that sleeps (~10s is the industry number).
2. Raise `terminationGracePeriodSeconds` to cover preStop **plus** in-flight
   request drain (≥ 15s here).
3. Deployment stays fully Available.

```sh
kubectl -n kubelings get deploy checkout-api -o jsonpath='{.spec.template.spec}' | head -c 400
```
