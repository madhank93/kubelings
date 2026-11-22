---
kind: unit
title: "get, describe, logs: finding out what happened"
name: read-the-cluster-unit
---

## The situation

From here on, most lessons start with something broken and no explanation.
Three commands do most of the finding out:

| Command | Answers |
|---|---|
| `kubectl get <kind>` | What exists, and is it healthy? One line per object. |
| `kubectl describe <kind> <name>` | Its full story: settings, conditions, and the **Events** at the bottom (scheduled, pulled, started, killed, failed). |
| `kubectl logs <pod>` | What the program itself printed to stdout and stderr. |

Add `-o wide` to `get` for extra columns such as the node, and `-o yaml` for
everything. `kubectl logs deploy/<name>` picks one of a Deployment's pods for
you.

A `greeter` app is running in `kubelings`. When it starts, it prints a random
secret word.

## Your task

Find two facts and record them in a ConfigMap named `findings`:

- `word`: the secret word `greeter` printed in its logs
- `node`: the name of the node `greeter`'s pod is running on

```sh
kubectl -n kubelings get pods -o wide
kubectl -n kubelings describe pod -l app=greeter
kubectl -n kubelings logs deploy/greeter
```

Then store what you found. The check reads it from here:

```sh
kubectl -n kubelings create configmap findings \
  --from-literal=word=<the-word> \
  --from-literal=node=<the-node>
```

<details>
<summary>Hint</summary>

The `NODE` column of `get pods -o wide` and the `Node:` line of `describe` both
give the node.

Got one wrong? A ConfigMap is replaced, not edited, the quick way:

```sh
kubectl -n kubelings delete configmap findings
kubectl -n kubelings create configmap findings --from-literal=word=... --from-literal=node=...
```

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

```sh
WORD=$(kubectl -n kubelings logs deploy/greeter | sed -n 's/^the secret word is //p')
NODE=$(kubectl -n kubelings get pods -l app=greeter -o jsonpath='{.items[0].spec.nodeName}')
kubectl -n kubelings create configmap findings --from-literal=word="$WORD" --from-literal=node="$NODE"
```

## Which command, when

- **Pod not Running?** `describe` first. The Events say whether it is waiting
  for a node, failing to pull its image, or being killed.
- **Pod Running but misbehaving, or restarting?** `logs`. Add `--previous` to
  read the logs of the container that just crashed.
- **Not sure where to look?** `kubectl get events -n <ns> --sort-by=.lastTimestamp`
  shows recent events for the whole namespace.

## Next

That is the toolkit. **Module 1 — Foundations** starts with a pod stuck in
`CrashLoopBackOff` and expects you to use exactly these three commands to find
out why.

</details>
