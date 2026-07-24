---
kind: unit
title: "static pods: the manifest the kubelet obeys directly"
name: static-pod-create-unit
---


> **☁ iximiuz Labs only.** A static pod is a file on a node that the kubelet
> runs without asking the API server — you need a real node, root, and
> `/etc/kubernetes/manifests` to do it. Not a kubectl-sandbox trick.

## The pod nobody scheduled

Everything you've run so far went through the front door: `kubectl` → API
server → scheduler picks a node → kubelet runs it. A **static pod** skips all of
that. The kubelet on a node watches a directory —
`/etc/kubernetes/manifests/` — and runs any pod manifest it finds there,
directly. No API server involved to *start* it. No scheduler to place it (it's
pinned to that node by definition). If the pod dies, that node's kubelet
restarts it.

You've been standing on static pods this whole module. `kube-apiserver`, `etcd`,
`kube-controller-manager`, `kube-scheduler` on a kubeadm control plane are all
static pods — which is exactly why 11.3 and 11.4 fixed the control plane by
*editing files*, not running `kubectl`. The kubelet can bring up the API server
because it doesn't need the API server to do it. That bootstrap chicken-and-egg
is the reason static pods exist.

## The mirror pod: how it shows up in kubectl

A static pod runs whether or not the API server is reachable — but once the API
*is* up, the kubelet publishes a **mirror pod**: a read-only copy of the static
pod, so it's visible to `kubectl get pods` and monitoring. Two tells make a
mirror pod recognisable:

- **Its name carries the node**: a static pod `web` on `node-02` appears as
  `web-node-02`. The kubelet appends the node name so mirrors from different
  nodes don't collide.
- **It's read-only and file-sourced.** `kubectl delete pod web-node-02` deletes
  only the mirror — the kubelet recreates it immediately from the file on disk
  (you saw this exact behaviour in the cloud-only cert-rotation lesson). Its
  annotation `kubernetes.io/config.source: file` marks where it really came
  from. To actually stop a static pod, you move its manifest out of the
  directory.

## Writing one

The manifest is an ordinary pod spec — the kubelet just wants a `Pod`:

```yaml
# /etc/kubernetes/manifests/web.yaml
apiVersion: v1
kind: Pod
metadata:
  name: web
  namespace: default
spec:
  containers:
  - name: web
    image: registry.k8s.io/pause:3.9
```

Save it and the kubelet notices within a second or two, starts the container,
and — as soon as it can reach the API — creates `web-node-02`. No `kubectl
apply`; the file *is* the source of truth. `crictl ps` on the node shows the
container running even before the mirror appears.

`registry.k8s.io/pause:3.9` is already on kubeadm nodes and starts instantly,
which makes it the reliable choice here; any image that starts and stays up
works.

## Your turn

On **node-02**, author a static pod:

1. Write a pod manifest to `/etc/kubernetes/manifests/web.yaml` — name `web`,
   namespace `default`, one container that starts and stays up.
2. Watch the kubelet pick it up: `crictl ps | grep web`.
3. Confirm the mirror `web-node-02` is `Running` in the API.

The check verifies `web-node-02` is `Running` **and** file-sourced — proving
it's a genuine static pod the kubelet owns, not a pod you created through the
API.

<details>
<summary>Hint</summary>

The file is the whole task — there's no `kubectl apply`. Drop a valid `Pod`
manifest into `/etc/kubernetes/manifests/web.yaml` on node-02 and the kubelet
runs it.

```sh
cat >/etc/kubernetes/manifests/web.yaml <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: web
  namespace: default
spec:
  containers:
  - name: web
    image: registry.k8s.io/pause:3.9
EOF
crictl ps | grep web         # kubelet started it directly
```

The mirror pod is named `web-node-02` (pod name + node name), not `web`. If you
tried `kubectl run web` instead, delete that pod — it isn't static — and place
the manifest in the directory so the kubelet, not the API, owns it.

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
# on node-02: write the manifest; the kubelet does the rest
cat >/etc/kubernetes/manifests/web.yaml <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: web
  namespace: default
spec:
  containers:
  - name: web
    image: registry.k8s.io/pause:3.9
EOF

# kubelet starts the container directly, no scheduler
crictl ps | grep web

# the mirror pod appears in the API as web-node-02
kubectl --kubeconfig=/etc/kubernetes/kubelet.conf -n default get pod web-node-02 -o wide
kubectl --kubeconfig=/etc/kubernetes/kubelet.conf -n default get pod web-node-02 \
  -o jsonpath='{.metadata.annotations.kubernetes\.io/config\.source}{"\n"}'   # -> file
```

To remove a static pod later, move its manifest out of the directory — deleting
the mirror pod won't do it, the kubelet just recreates it from the file.

</details>

## Root cause, restated

There's nothing broken here — this is the mechanism the rest of the module
stands on, learned by hand.

- **Static pods bypass the scheduler and API.** The kubelet runs any manifest
  in `/etc/kubernetes/manifests/` directly and pins it to that node. That's how
  the control plane boots itself.
- **The mirror pod is a read-only shadow.** Named `<pod>-<node>`, annotated
  `config.source: file`. You can see it and delete it in the API, but the file
  on disk is the real source — delete the mirror and it comes right back.
- **To stop a static pod, move its file.** Everything in this module that
  "restarted the control plane" did it by moving manifests out of and back into
  that directory — because that's the only handle the API doesn't have.
