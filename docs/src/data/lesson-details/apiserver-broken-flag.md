> **☁ iximiuz Labs only.** You're going to take the API server down and bring
> it back — editing a static-pod manifest on the control-plane host and reading
> container logs through `crictl` while `kubectl` is dead. That needs a real
> control plane you own, root, and the runtime beneath it.

## The task the exam always includes

Somewhere in a CKA sitting you get a cluster where `kubectl` just hangs, and
the task is: fix it. It's the most disorienting failure in the whole exam
because your primary tool is the thing that's broken — no `kubectl get`, no
`kubectl describe`, no events. You have to diagnose the control plane from
*under* the control plane.

The good news: `kube-apiserver` on a kubeadm cluster is a **static pod** (7.4),
and static pods fail loudly and locally. The kubelet reads
`/etc/kubernetes/manifests/kube-apiserver.yaml`, runs it, and — when it
crashes — keeps recreating it in a tight loop. Everything you need is on the
node: the manifest that defines it, and the runtime logs that say why it died.

## Diagnosing with the API down

The manifest lists the API server's entire command line as YAML. A single bad
flag — an unknown one, or a valid one with an invalid value — and the process
exits at startup before it ever binds a port. Because it's a static pod, no
Deployment or ReplicaSet is involved; the kubelet alone owns its lifecycle.

With no API, you read the container directly:

```sh
# the API server's containers, including exited ones (-a)
crictl ps -a | grep kube-apiserver

# the logs of the most recent attempt — the startup error is right at the top
crictl logs <container-id>
```

That log line is the whole answer: `unknown flag: --…`, or `invalid value
"…" for flag --…`, or a cert path it can't read. You don't guess — the runtime
already caught the process's dying words.

## Fixing it and letting the kubelet rebuild

You fix a static pod by editing its manifest in place. Save the corrected
`kube-apiserver.yaml` and the kubelet notices the change, tears down the broken
container, and starts a fresh one from the new spec — no `kubectl`, no restart
command, because the kubelet is already watching that directory (7.4).

Two habits that save you here:

- **Back up before you edit.** `cp kube-apiserver.yaml /root/` first, so a
  fat-fingered YAML edit doesn't turn one broken flag into a broken file.
- **Watch it come back with `crictl`, not `kubectl`.** `crictl ps` shows the
  new container going `Running`; only once `/readyz` passes will `kubectl`
  answer again. Give the kubelet ~20 seconds after your save.

Mind the YAML: the flags are a list under `command:`, each its own `- --flag`
line at a fixed indentation. Remove the whole offending line, and don't disturb
the indentation of its neighbours — a broken manifest won't even be parsed, and
the kubelet will just skip it.

## Your turn

`init` added a flag `kube-apiserver` doesn't understand to the manifest on
**cplane-01**. The API server is crash-looping and `kubectl` is timing out.

Bring it back:

1. On **cplane-01**, read why it's dying — `crictl ps -a | grep
   kube-apiserver`, then `crictl logs <id>`.
2. Edit `/etc/kubernetes/manifests/kube-apiserver.yaml` and remove the bad
   flag. (A pristine copy is at `/root/kubelings-apiserver-baseline.yaml` if
   you need it.)
3. Wait for the kubelet to rebuild the static pod and confirm the API is
   serving — `crictl ps`, then `kubectl get nodes`.

The check confirms the bad flag is gone from the manifest **and** that
`/readyz` passes — a manifest edit that doesn't bring the API back doesn't
count.
