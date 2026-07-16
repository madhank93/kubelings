> **Reading.** CNI lives in the node's filesystem — `/etc/cni/net.d/`,
> `/opt/cni/bin/` — outside the kubectl sandbox, so this is a guided tour
> with the exact commands for when you have a node (any kind cluster:
> `docker exec -it <node> ls /etc/cni/net.d/`). It also satisfies the M7
> backlog's "CNI hands-on" reading half — the kubelet side of the story.

## The gap nobody tells you about

Vanilla Kubernetes **cannot connect two pods.** The kubelet creates a pod's
network namespace and then — hands the problem to whatever binary it finds
configured under the **Container Network Interface**: give the pod an IP,
wire it to every other pod, tell me what you did. No CNI plugin installed?
Every new pod hangs in `ContainerCreating`, and the node itself reports:

```
NotReady   ... container runtime network not ready:
NetworkReady=false reason:NetworkPluginNotReady
```

That message is the CNI's absence speaking. You've met its cousins already:
`kubeadm init` finishes with nodes NotReady *by design* until you apply a
CNI manifest.

## The two directories that are the whole interface

```sh
# on any node (kind: docker exec -it kind-control-plane bash):
ls /etc/cni/net.d/            # 10-kindnet.conflist — the CONFIG
ls /opt/cni/bin/              # bridge, ptp, loopback, host-local, ... — the BINARIES
```

The **conflist** is a JSON chain of plugins, executed in order:

```json
{
  "cniVersion": "0.3.1",
  "name": "kindnet",
  "plugins": [
    { "type": "ptp",
      "ipam": { "type": "host-local",
                "ranges": [[{ "subnet": "10.244.1.0/24" }]] } },
    { "type": "portmap", "capabilities": {"portMappings": true} }
  ]
}
```

Read it like a sentence: *use the `ptp` binary to create the veth pair, let
`host-local` hand out IPs from this node's `10.244.1.0/24` slice, then let
`portmap` handle hostPorts.* Each `"type"` names a binary in `/opt/cni/bin/`
— that's the entire discovery mechanism. Lowest-numbered file in
`/etc/cni/net.d/` wins; a stale second file there is a classic breakage.

Who calls it? Not the kubelet directly: kubelet → CRI (containerd) → CNI.
containerd execs the plugin with the pod's netns path in env vars; the plugin
prints the resulting IP as JSON to stdout. It's Unix all the way down.

## What Calico and Flannel actually are

A "CNI install" (`kubectl apply -f calico.yaml`) is just a **DaemonSet** whose
init containers copy plugin binaries into `/opt/cni/bin/` and write the
conflist into `/etc/cni/net.d/` — plus a long-running agent per node:

- **Flannel**: simplicity — one VXLAN overlay, `flanneld` programs the vtep;
  pod CIDR slices come from the Node objects.
- **Calico**: routing — BGP (or VXLAN) with `bird`/`felix` programming real
  routes and, crucially, **NetworkPolicy enforcement** via iptables/eBPF.

That last point is why this course's local kind clusters verify NetworkPolicy
*objects* but don't enforce them: kindnet wires pods but implements no policy
engine. The API server stores whatever policy YAML you like; only the CNI
decides whether it means anything. Same YAML, different CNI, different
cluster behavior — remember that when a policy "works in prod but not in
kind" (or the reverse).

This is also why **the Datadog incident** (M8.6) was a CNI story: an OS
update under Cilium's feet took out the dataplane on 31k nodes. The CNI is a
per-node system daemon, and it fails like one.

## Triage: pod stuck ContainerCreating

```sh
kubectl describe pod <name>        # Events: the only place the truth lives
```

| Event says | It's | Go |
|---|---|---|
| `NetworkPluginNotReady` / node NotReady | CNI dead or absent | `kubectl -n kube-system get ds` — is the CNI DaemonSet running on that node? crashlooping? |
| `failed to allocate for range` / `no IP addresses available` | IPAM exhausted — node's pod CIDR slice is full | count pods per node vs `--max-pods`; look for leaked IP reservations in `/var/lib/cni/` |
| `error getting ClusterInformation: connection refused` (Calico) | agent can't reach its datastore | the CNI DaemonSet's own logs |
| pod is Running but traffic doesn't flow | **not the CNI** — policy or Service wiring | `networkpolicy-blackhole` (4.2) path: check policies selecting the pod, then endpoints (4.3) |

The decision that matters: **ContainerCreating = plumbing (CNI), Running but
unreachable = policy or Service.** Don't debug iptables when the pod has no
IP; don't reinstall Calico when the pod has one.

On a node, the CRI view beats guessing: `crictl pods` / `crictl ps` show what
the runtime actually created, and `crictl inspectp <pod>` shows the netns and
IP the CNI reported — or the error it didn't survive.

## The 20-second mental model

```
kubelet ──► containerd ──► /opt/cni/bin/<type>   (per conflist order)
                              │ reads /etc/cni/net.d/*.conflist
                              ▼
                    veth pair + IP from this node's CIDR slice
```

- No conflist → node NotReady, pods ContainerCreating. Forever.
- Plugins are per-node files delivered by a DaemonSet; the "CNI" you chose
  is that DaemonSet plus opinions about routing and policy.
- NetworkPolicy is a CNI feature, not a Kubernetes one — the API only stores
  the objects (4.2 taught the blackhole; the enforcement engine is what
  differs per CNI).
- kubelet's `--max-pods` (default 110) and the per-node CIDR size must agree
  — a /24 slice holds 254 IPs, comfortable; smaller slices are how "no IP
  addresses available" pages at 3 a.m. (Neon's 2024 IP-exhaustion incident
  in the [library](https://kubelings.madhan.app/reference/incident-library/)
  is this at cloud scale.)
