## The situation

Kernel patch night. Runbook step 3: `kubectl drain kubelings-worker`. You run it
and watch:

```
evicting pod kubelings/tickets-6d8f7b9c4-2xkpl
error when evicting pods/"tickets-6d8f7b9c4-2xkpl" -n "kubelings" (will retry after 5s):
Cannot evict pod as it would violate the pod's disruption budget.
```

…and again. And again. Ten minutes of the same two lines. The drain isn't
broken — it's being *refused*, politely, forever.

A **PodDisruptionBudget** is a contract with the eviction API: *"never let
voluntary disruptions take availability below X."* Drains, node upgrades,
cluster-autoscaler scale-downs — all go through eviction and all must honor it.

Now the math on this one:

```
replicas:      2
minAvailable:  2
disruptionsAllowed = 2 - 2 = 0
```

Zero. Not "wait until it's safer" — **structurally zero, at all times**. Whoever
wrote this PDB demanded 100% of replicas be up always, which outlaws
maintenance itself.

## Your task

Make maintenance possible without giving up protection:

1. Look at what the PDB currently allows (`kubectl get pdb` shows the columns).
2. Fix the contract so at least one disruption is allowed while the app stays
   protected — change the PDB's math, or give it more replicas to budget with.
   Keep a PDB either way.
3. `tickets` must remain fully Available.

```sh
kubectl -n kubelings get pdb tickets-pdb
kubectl -n kubelings get deploy tickets
```
