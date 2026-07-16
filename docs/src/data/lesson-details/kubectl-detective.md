## The situation

The alert reads, in its entirety: **"1 of 5 services degraded."**

Which one? The alert doesn't say. The dashboard that would say is owned by a team
in another timezone. You have `kubectl`, the `kubelings` namespace, and five
deployments: `catalog`, `checkout`, `payments`, `search`, `recommendations`.

This is the daily reality of cluster operations: not exotic failures, but *"which
of these many identical-looking things is the broken one?"* The engineers who
resolve these in minutes aren't luckier — they run the same short loop every
time, widest view first, narrowing on anomaly:

```
get (wide, everything) → spot the odd number → describe it → events/logs → fix
```

## Your task

Find the degraded service and restore it. Every deployment must end with at
least one available replica.

Start wide:

```sh
kubectl -n kubelings get deploy          # READY column — read it like a detective
kubectl -n kubelings get pods
kubectl -n kubelings get events --sort-by=.lastTimestamp | tail -15
```
