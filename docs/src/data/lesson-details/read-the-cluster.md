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
