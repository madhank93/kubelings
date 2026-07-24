---
kind: unit
title: "Audit the cluster: CIS benchmark with kube-bench"
name: cis-kube-bench-unit
---


## The situation

Everything so far in this module hardened *workloads*. The pen-test report's
first section, though, will be about the *platform*: is the API server
accepting anonymous requests? Are kubelet ports open? Are etcd's files
world-readable? The industry checklist for those questions is the
**CIS Kubernetes Benchmark** — a few hundred audited controls for every
control-plane and node component — and **kube-bench** is the tool that runs
it *as a pod on the cluster it's auditing*.

Why a pod can audit the node at all: the checks read component config files
and process flags, so the Job mounts the node's config directories
(`hostPath`, read-only) and runs with the node's PID namespace. That's a lot
of trust — which is itself the first lesson: **an auditor pod is exactly the
shape of pod your Pod Security policy (6.4) exists to block.** It runs here
because *you*, the admin, deliberately grant it.

## Your task

Run the benchmark as a Job in `kubelings` and read its verdicts:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: kube-bench
  namespace: kubelings
spec:
  backoffLimit: 1
  template:
    spec:
      hostPID: true
      restartPolicy: Never
      containers:
        - name: kube-bench
          image: docker.io/aquasec/kube-bench:latest
          command: ["kube-bench"]
          volumeMounts:
            - {name: var-lib-kubelet, mountPath: /var/lib/kubelet, readOnly: true}
            - {name: etc-systemd, mountPath: /etc/systemd, readOnly: true}
            - {name: etc-kubernetes, mountPath: /etc/kubernetes, readOnly: true}
            - {name: usr-bin, mountPath: /usr/local/mount-from-host/bin, readOnly: true}
      volumes:
        - {name: var-lib-kubelet, hostPath: {path: /var/lib/kubelet}}
        - {name: etc-systemd, hostPath: {path: /etc/systemd}}
        - {name: etc-kubernetes, hostPath: {path: /etc/kubernetes}}
        - {name: usr-bin, hostPath: {path: /usr/bin}}
      tolerations:
        - {key: node-role.kubernetes.io/control-plane, operator: Exists, effect: NoSchedule}
```

Save as `kube-bench-job.yaml`, apply, wait, read:

```sh
kubectl apply -f kube-bench-job.yaml
kubectl -n kubelings wait --for=condition=complete job/kube-bench --timeout=180s
kubectl -n kubelings logs job/kube-bench | less     # the actual deliverable
```

<details>
<summary>Hint</summary>

The job lands on a worker by default, so you'll get the **node** checks
(section 4: kubelet, kube-proxy). Skim for lines starting `[FAIL]` and
`[WARN]`, then find the same numbered item under `== Remediations ==` — every
finding ships with its fix. Count the damage:

```sh
kubectl -n kubelings logs job/kube-bench | grep -c '\[FAIL\]'
kubectl -n kubelings logs job/kube-bench | grep '== Summary' -A5
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


## Reading a kube-bench report like an operator

Every check is `[PASS]`, `[FAIL]`, `[WARN]`, or `[INFO]` against a numbered
CIS control, grouped by component:

| Section | Component | The classics that fail |
|---|---|---|
| 1 | API server, controller-manager, scheduler | `--anonymous-auth` not false, audit logging off, `--profiling` on |
| 2 | etcd | data dir permissions, peer TLS |
| 4 | kubelet | `--anonymous-auth`, `authorization-mode=AlwaysAllow`, readOnlyPort 10255 open |
| 5 | policies | RBAC wildcards, default SA automount, no PSS — **all lessons in this module** |

Triage rules that keep this useful instead of overwhelming:

- **FAIL on auth-related kubelet/API flags = fix first.** Anonymous kubelet
  access is remote code execution on the node, full stop.
- **WARN often means "manual check"** — kube-bench couldn't verify
  mechanically. Don't skip them; they hide the policy-level items.
- **Not every FAIL is yours to fix.** On managed clusters (EKS/GKE/AKS) the
  control plane isn't reachable — use the provider-specific benchmark
  variants and own the node + policy sections. On this kind cluster some
  findings are kind's dev-cluster conveniences; a real cluster must not
  inherit them.
- Section 5 findings should read like this module's table of contents: RBAC
  wildcards (6.1), SA automount (6.5), missing PSS (6.4). The benchmark is
  the audit trail proving you did those lessons.

## Where the remediations happen

kube-bench *finds*; fixing means editing component config — API server flags
live in `/etc/kubernetes/manifests/kube-apiserver.yaml` (a static pod, M7),
kubelet config in `/var/lib/kubelet/config.yaml` — host-level changes outside
this course's kubectl-only sandbox, done via your node image or kubeadm
config in real fleets (see the control-plane hardening reading next).

## Prevention

- Run kube-bench **on a schedule** (CronJob, M2.5) and alert on the FAIL
  count rising — config drift applies to control planes too (M3.6's lesson,
  one layer down).
- Pin the kube-bench image and benchmark version in real use, so audit
  results are comparable release to release.
- Treat the report as CI for infrastructure: new node image → run benchmark →
  diff findings → merge. Auditors love it; more importantly, it's true.

</details>
