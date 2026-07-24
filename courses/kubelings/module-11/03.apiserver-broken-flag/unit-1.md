---
kind: unit
title: "you broke the control plane: read the crash without an API"
name: apiserver-broken-flag-unit
---


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

<details>
<summary>Hint</summary>

You have no API, so don't reach for `kubectl`. Everything is on cplane-01:

```sh
crictl ps -a | grep kube-apiserver     # find the crash-looping container
crictl logs <container-id>             # the first lines name the bad flag
```

The flag lives in `/etc/kubernetes/manifests/kube-apiserver.yaml`, under
`spec.containers[0].command`, as a `- --…` list item. Delete that one line,
keep every other line's indentation exactly as it was, and save.

The kubelet is watching that directory — it rebuilds the pod on save, no
restart needed. Watch `crictl ps` until the container is `Running`, then wait a
few more seconds for `/readyz`; `kubectl` starts answering the moment the API
is ready.

If you damaged the file, copy the baseline back and remove only the bad flag
from that clean copy:

```sh
cp /root/kubelings-apiserver-baseline.yaml /etc/kubernetes/manifests/kube-apiserver.yaml
# then delete the '- --kubelings-invalid-flag=true' line
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
# 1 · no API — read the crash from the runtime
crictl ps -a | grep kube-apiserver
crictl logs "$(crictl ps -a --name kube-apiserver -q | head -1)"
#   -> "unknown flag: --kubelings-invalid-flag"

# 2 · remove the bad flag from the static-pod manifest (surgical sed, or an editor)
sed -i '/--kubelings-invalid-flag=true/d' /etc/kubernetes/manifests/kube-apiserver.yaml

# 3 · the kubelet rebuilds the pod on save; watch it, then the API
crictl ps | grep kube-apiserver                 # new container -> Running
until kubectl --kubeconfig=/etc/kubernetes/admin.conf get --raw=/readyz >/dev/null 2>&1; do
  sleep 2
done
kubectl --kubeconfig=/etc/kubernetes/admin.conf get nodes
```

The `sed` deletes just the offending line and leaves the rest of the manifest
untouched — the surgical version of "open the file and remove one line." If the
file was damaged, restore `/root/kubelings-apiserver-baseline.yaml` first and
delete the bad line from that clean copy.

</details>

## Root cause, restated

The control plane is data (manifests) plus processes (static pods), and both
live on the node. When `kubectl` dies, you diagnose one level down.

- **The API server is a static pod.** Its command line is YAML in
  `/etc/kubernetes/manifests/`; the kubelet owns it and rebuilds it on any edit
  — no Deployment, no `kubectl` needed to restart it.
- **`crictl logs` is how you debug with no API.** When the control plane is
  down, the runtime still has the container and its dying log line. `crictl ps
  -a` to find it, `crictl logs` to read it.
- **A bad flag exits at startup, before binding.** That's why the symptom is a
  *hang*, not an error — nothing is listening. The fix is always in the
  manifest, and the API returns only when `/readyz` passes, not the instant you
  save.
