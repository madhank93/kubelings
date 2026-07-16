> **Reading.** Cluster API needs a management cluster plus infrastructure
> to provision (cloud credentials, or Docker for the dev provider) — beyond
> this playground. The YAML below is real and complete; try it at home with
> `clusterctl` + the Docker provider (CAPD) on any kind cluster.

## The idea, in one sentence

You bootstrapped a cluster by hand in M7.10 — `kubeadm init`, tokens, CNI,
joins. **Cluster API (CAPI)** asks: what if that whole ceremony were a
controller's job, and a *cluster* were just a custom resource you apply?

```
Deployment  → ReplicaSet   → Pods      (M2: the pattern you know)
MachineDeployment → MachineSet → Machines → actual VMs running kubelets
Cluster     → control plane + machines  (the same pattern, one level up)
```

It's M7.5's operator pattern (CRDs + reconcile loops) pointed at
infrastructure: desired state "a 1.31 cluster with 3 workers", observed
state "what exists", and controllers converging the difference — including
*deleting and recreating machines* to get there.

## The cast

One **management cluster** runs the CAPI controllers and holds the CRDs;
it reconciles many **workload clusters** (which don't run CAPI at all —
they're the product, not the machinery):

| Object | Analog | Owns |
|---|---|---|
| `Cluster` | the umbrella | networking CIDRs, refs to the two below |
| `KubeadmControlPlane` | a StatefulSet-of-control-planes | CP replica count, k8s version, kubeadm config (M7.10's init flags, as spec) |
| `MachineDeployment` | Deployment | worker replicas, version, template |
| `MachineSet` / `Machine` | ReplicaSet / Pod | one Machine ⇄ one node |
| `*MachineTemplate` (provider) | pod template | instance type, image, disk |

**Providers** are the pluggable half — each is its own operator:
*infrastructure* providers (AWS/Azure/GCP/vSphere/Docker) turn Machine
into an actual VM; the *bootstrap* provider renders cloud-init that runs —
literally — `kubeadm init`/`join` (M7.10 is inside the machinery, not
replaced by it); the *control-plane* provider orchestrates CP scaling the
way M7.11 described doing by hand.

## What the YAML looks like

Workers for a cluster on the Docker provider — read it as "a Deployment
whose pods are nodes":

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  name: prod-workers
spec:
  clusterName: prod
  replicas: 3
  selector:
    matchLabels: {cluster.x-k8s.io/cluster-name: prod}
  template:
    spec:
      clusterName: prod
      version: v1.31.4
      bootstrap:
        configRef:
          kind: KubeadmConfigTemplate        # renders the kubeadm join
          name: prod-workers
      infrastructureRef:
        kind: DockerMachineTemplate          # provider: what a "machine" is
        name: prod-workers
```

Every Deployment verb transfers: scale workers with `kubectl scale
machinedeployment prod-workers --replicas=5`; upgrade Kubernetes by
bumping `version:` — CAPI rolling-replaces machines (cordon, drain — the
M8.8 cycle, automated — then delete, then a fresh machine joins). The
M2.19 rolling-update intuitions apply to *nodes*.

```sh
# on a management cluster, the fleet reads like workloads:
kubectl get clusters,kubeadmcontrolplanes,machinedeployments,machines
# NAME       PHASE         VERSION   REPLICAS   READY
# prod       Provisioned
# prod-cp    …             v1.31.4   3          3
# prod-workers             v1.31.4   3          3
```

`clusterctl` (the CLI) initializes the management cluster
(`clusterctl init --infrastructure docker`) and generates starter YAML —
after which everything is kubectl and, naturally, GitOps: cluster
definitions in git, Argo CD/Flux (10.1–10.3) syncing them. Clusters become
commits.

## CAPI vs kubeadm-by-hand

| | kubeadm (M7.10) | Cluster API |
|---|---|---|
| Creates a cluster | ✅ you, per machine | ✅ controller, from YAML |
| Repairs a dead node | you notice, you replace | MachineHealthCheck auto-remediates |
| Upgrades | M8.7 runbook, node by node | bump `version:`, rolling machine replacement |
| Fleet of 40 clusters | 40 runbooks | 40 YAML files and one management cluster |
| Cost | a runbook | a management cluster to operate — which is itself a SPOF to design for (M7.11 thinking, one level up) |

Managed offerings (EKS/GKE/AKS) occupy the middle: the provider runs the
control plane, and CAPI has providers to drive *them* too — one API over
heterogeneous fleets is the actual selling point.

## Takeaway

- CAPI = the operator pattern applied to clusters: management cluster,
  provider operators, Machines reconciled like pods.
- The abstraction ladder: `Cluster → KubeadmControlPlane +
  MachineDeployment → MachineSet → Machine → VM` — every level a CRD you
  can `kubectl get`.
- kubeadm didn't go away; CAPI runs it for you via bootstrap configs.
  Everything M7.10/7.11 taught is what the controllers are doing.
- Node upgrades become rolling *machine replacements* — immutable
  infrastructure, no in-place patching, maintenance (M8.8) as a controller
  behavior.
- Try it for real: kind cluster + `clusterctl init --infrastructure
  docker` — a cluster that creates clusters, on your laptop.
