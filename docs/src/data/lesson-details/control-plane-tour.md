> **Guided reading + live pokes.** Run the commands on this
> cluster as you go, then answer the quick check at the end. This ties
> together reconcile (7.1), scheduler (7.2), and etcd (7.3) into the whole
> picture.

## Part 1 — the life of `kubectl apply -f pod.yaml`

Follow one request end to end:

1. **kubectl** turns your YAML into an HTTP `POST /api/v1/namespaces/…/pods`.
2. **API server — authentication:** who are you? (client cert, token, OIDC).
   ServiceAccount tokens (Module 6) authenticate here.
3. **API server — authorization:** may you? RBAC evaluates your verbs/resources
   (the `kubectl auth can-i` you practiced *is* this stage, exposed).
4. **API server — admission:**
   - **Mutating** webhooks/plugins rewrite the object (inject sidecars, defaults,
     the Pod Security *warn*).
   - Schema **validation**.
   - **Validating** webhooks/plugins accept or reject (the Jetstack incident
     lived here; Pod Security *enforce* lives here).
5. **Persist to etcd:** the object is written under `/registry/...` (7.3). Only
   now does `kubectl` get its `201 Created`.
6. **Asynchronously, everyone reacts to the watch event:** scheduler binds a node
   (7.2), kubelet on that node starts containers (Part 2), controllers update
   status. The write path is synchronous; the *convergence* is eventual.

See admission and RBAC narrate themselves:

```sh
kubectl auth can-i create pods -n kubelings
kubectl -n kubelings run tour --image=nginx:1.27-alpine --dry-run=server -o yaml | head -30
#   ^ --dry-run=server runs steps 2–4 (auth, admission, defaulting) WITHOUT step 5
```

`--dry-run=server` is the whole pre-etcd pipeline with the persist removed — the
best way to *see* mutation/defaulting happen.

## Part 2 — the kubelet and the CRI

The scheduler wrote `nodeName`. Now the **kubelet** on that node takes over — it
is the reconcile loop *for its node*: "make the containers match the PodSpecs
assigned to me."

It doesn't run containers itself. It calls the **Container Runtime Interface
(CRI)** — a gRPC API — against the node's runtime (containerd on kind). The
runtime pulls images and asks the kernel (namespaces, cgroups) to create
containers. Peek on a node:

```sh
# on a kind node (container): the runtime's own view, underneath Kubernetes
docker exec kubelings-worker crictl ps 2>/dev/null | head
docker exec kubelings-worker crictl pods 2>/dev/null | head
```

`crictl` talks straight to containerd — the containers Kubernetes created, seen
below Kubernetes. Note the **pause** container per pod: it holds the shared
network namespace so a pod's containers share one IP (the plumbing behind every
Service lesson). Networking itself is handed to a **CNI** plugin at pod creation;
storage mounts to a **CSI** plugin. kubelet orchestrates; pluggable interfaces do
the work — the same "pluggable pipeline" pattern as the scheduler.

## Part 3 — leader election (one brain, many replicas)

`kube-controller-manager` and `kube-scheduler` run as multiple replicas for HA —
but if *all* replicas ran the reconcile loop, they'd fight (three controllers
each creating a replacement pod = chaos). So exactly one is active at a time, via
**leader election** over a `Lease` object:

```sh
kubectl -n kube-system get lease
kubectl -n kube-system get lease kube-scheduler -o jsonpath='{.spec.holderIdentity}{"\n"}'
```

The holder must **renew** the lease every few seconds. Stop renewing (crash,
partition, `NotReady`) → the lease expires → a standby acquires it and becomes
active. That's failover: no orchestrator, just a timed lock in the API. The same
`Lease`/lock pattern is how *you'd* make a custom controller or operator HA.

## The whole picture, one paragraph

**kubectl** talks to the **API server**, which authenticates, authorizes, runs
**admission**, and persists to **etcd** — the only state. Everything else —
**scheduler**, **controller-manager** (via elected **leader**), **kubelet**
(via **CRI/CNI/CSI**) — is a controller **watching** the API server and
**reconciling** its slice of the world toward the spec in etcd. Stateless facade,
single source of truth, many independent loops. Internalize this and no
Kubernetes behavior is a black box again — including every failure in the
[Incident Library](https://kubelings.madhan.app/reference/incident-library/),
each of which is one of these components doing exactly what it was told.

## Quick check

Record your answers in a ConfigMap named `quick-check` in the `kubelings`
namespace: one key per question, the letter as the value.

1. When does kubectl get its 201 Created for a new pod?

   - **a)** After the object is persisted to etcd, before any node runs it
   - **b)** After the scheduler binds it to a node
   - **c)** After its containers are running

2. What does `kubectl ... --dry-run=server` run?

   - **a)** Only client-side schema validation
   - **b)** Authentication, authorization and admission, without persisting to etcd
   - **c)** The whole pipeline, then deletes the object

3. Three kube-scheduler replicas run for HA. How many schedule pods at once?

   - **a)** All three, sharing the work
   - **b)** Two, with one standby
   - **c)** One: the replica holding the kube-scheduler Lease

```sh
kubectl -n kubelings create configmap quick-check \
  --from-literal=q1=<letter> \
  --from-literal=q2=<letter> \
  --from-literal=q3=<letter>
```

Wrong answer? `kubectl -n kubelings delete configmap quick-check` and create it again.
